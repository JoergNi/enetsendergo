package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "time/tzdata"
)

var (
	version       = "dev"     // overridden by -ldflags "-X main.version=..."
	gitCommit     = "unknown" // overridden by -ldflags "-X main.gitCommit=..."
	lastInitTime  time.Time
	jobs          []*Job
	jobsMu        sync.Mutex
	dailyHighTemp *float64
	weather       *WeatherService

	schedulerWaitingNTP atomic.Bool

	firmwareVersion = "unknown"
	hardwareVersion = "unknown"
	enetVersion     = "unknown"
	deviceTypes     []int
)

const hotThresholdCelsius = 24

func main() {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		panic("could not load Europe/Berlin timezone: " + err.Error())
	}
	time.Local = loc

	host := GetConfig("enet_host", "ENET_HOST", "192.168.178.34")
	port := 9050
	if p, err := strconv.Atoi(GetConfig("enet_port", "ENET_PORT", "9050")); err == nil {
		port = p
	}

	sender := NewSocketSender(host, port)
	InitRegistry(sender)

	axiomDataset := GetConfig("axiom_dataset", "AXIOM_DATASET", "")
	axiomToken := GetConfig("axiom_api_token", "AXIOM_API_TOKEN", "")
	if axiomDataset != "" && axiomToken != "" {
		aw := newAxiomBatchWriter(axiomDataset, axiomToken, "enetsender")
		aw.start(context.Background())
		axiomWriter = aw
	}

	OnCommandFailed = func(msg string) { LogNormal("[FAIL] " + msg) }
	LogNormal(fmt.Sprintf("[START] eNet Sender %s (commit=%s) starting", version, gitCommit))
	if axiomWriter != nil {
		LogNormal(fmt.Sprintf("[START] Axiom logging enabled (dataset=%s)", axiomDataset))
	}

	queryVersion()
	queryAllChannels()
	logThingStates()

	weather = NewWeatherService(sunLat, sunLon, nil)

	go runScheduler()
	go refreshStateCache()
	go heartbeatLoop()
	go weatherFetchLoop()

	startHTTPServer()
}

func queryVersion() {
	response := Registry.OfficeGarage.SendRequest(NewVersionRequest(), 2000)
	LogDebug("VERSION_RES: " + strings.TrimSpace(response))
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(response)), &parsed); err == nil {
		if v, ok := parsed["FIRMWARE"].(string); ok {
			firmwareVersion = v
		}
		if v, ok := parsed["HARDWARE"].(string); ok {
			hardwareVersion = v
		}
		if v, ok := parsed["ENET"].(string); ok {
			enetVersion = v
		}
	} else {
		LogNormal("Failed to parse VERSION_RES")
	}
}

func queryAllChannels() {
	response := Registry.OfficeGarage.SendRequest(NewGetChannelInfoAllRequest(), 2000)
	LogDebug("GET_CHANNEL_INFO_ALL: " + strings.TrimSpace(response))
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(response)), &parsed); err == nil {
		if devArr, ok := parsed["DEVICES"].([]interface{}); ok {
			deviceTypes = make([]int, len(devArr))
			for i, v := range devArr {
				if f, ok := v.(float64); ok {
					deviceTypes[i] = int(f)
				}
			}
		}
	} else {
		LogNormal("Failed to parse GET_CHANNEL_INFO_ALL_RES")
	}
}

func logThingStates() {
	for _, thing := range Registry.All {
		state := thing.GetState()
		if state != nil {
			LogDebug(fmt.Sprintf("[Ch%02d] %-30s Value=%3d  State=%s", thing.Channel, thing.Name, state.Value, state.State))
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func heartbeatLoop() {
	for {
		time.Sleep(60 * time.Second)
		suffix := ""
		if schedulerWaitingNTP.Load() {
			suffix = " scheduler=waiting-for-ntp-sync"
		}
		LogNormal(fmt.Sprintf("heartbeat - things=%d lastInit=%s jobs=%d%s",
			len(Registry.All), lastInitTime.Format("15:04:05"), len(jobs), suffix))
	}
}

func weatherFetchLoop() {
	for {
		if dailyHighTemp == nil {
			temp := weather.GetDailyHighTemperature()
			if temp != nil {
				dailyHighTemp = temp
				LogNormal(fmt.Sprintf("Today's high: %.1f°C", *temp))
			} else {
				time.Sleep(10 * time.Minute)
				continue
			}
		}
		now := time.Now()
		nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		time.Sleep(time.Until(nextMidnight))
		dailyHighTemp = nil
	}
}

func runScheduler() {
	schedulerWaitingNTP.Store(true)
	waitForNTPSync()
	schedulerWaitingNTP.Store(false)
	for {
		if lastInitTime.Day() != time.Now().Day() || lastInitTime.IsZero() {
			initialize()
		}
		jobsMu.Lock()
		for _, j := range jobs {
			j.Check()
		}
		jobsMu.Unlock()
		time.Sleep(5 * time.Second)
	}
}

func refreshStateCache() {
	refreshing := make(map[int]bool)
	var refreshMu sync.Mutex

	for {
		for _, thing := range Registry.All {
			state := thing.GetState()
			if state != nil && state.Value >= 0 {
				cached, ok := StateCache.Get(thing.Channel)
				changed := !ok || cached.Value != state.Value || cached.State != state.State
				StateCache.Set(thing.Channel, state)

				if changed {
					refreshMu.Lock()
					if !refreshing[thing.Channel] {
						refreshing[thing.Channel] = true
						ch := thing.Channel
						t := thing
						go func() {
							refreshChannel(t)
							refreshMu.Lock()
							delete(refreshing, ch)
							refreshMu.Unlock()
						}()
					}
					refreshMu.Unlock()
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
		time.Sleep(60 * time.Second)
	}
}

func refreshChannel(thing *Thing) {
	initial := thing.GetState()
	if initial != nil && initial.Value >= 0 {
		StateCache.Set(thing.Channel, initial)
	}

	lastKey := ""
	if initial != nil {
		lastKey = fmt.Sprintf("%d:%s", initial.Value, initial.State)
	}
	unchanged := 0
	for unchanged < 2 {
		time.Sleep(3 * time.Second)
		state := thing.GetState()
		if state == nil {
			break
		}
		if state.Value >= 0 {
			StateCache.Set(thing.Channel, state)
			key := fmt.Sprintf("%d:%s", state.Value, state.State)
			if key == lastKey {
				unchanged++
			} else {
				unchanged = 0
			}
			lastKey = key
		}
	}
}

// minTime returns the earlier of a and today+hour. today must be midnight of the current day.
func minTime(a time.Time, today time.Time, hour float64) time.Time {
	b := today.Add(time.Duration(hour * float64(time.Hour)))
	if a.Before(b) {
		return a
	}
	return b
}

// maxTime returns the later of a and today+hour. today must be midnight of the current day.
func maxTime(a time.Time, today time.Time, hour float64) time.Time {
	b := today.Add(time.Duration(hour * float64(time.Hour)))
	if a.After(b) {
		return a
	}
	return b
}

// Vacation override 2026-08-08 through 2026-08-22: living-room raffstores must not move up before 16:00 local time.
func livingRoomRaffstoresUpTime(now time.Time, today time.Time, sunrise time.Time) time.Time {
	currentDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startDate := time.Date(2026, time.August, 8, 0, 0, 0, 0, now.Location())
	endDate := time.Date(2026, time.August, 22, 0, 0, 0, 0, now.Location())
	earliestHour := 10.0
	if !currentDate.Before(startDate) && !currentDate.After(endDate) {
		earliestHour = 16
	}
	return maxTime(sunrise.Add(10*time.Minute), today, earliestHour)
}

func initialize() {
	lastInitTime = time.Now()
	today := time.Date(lastInitTime.Year(), lastInitTime.Month(), lastInitTime.Day(), 0, 0, 0, 0, lastInitTime.Location())
	LogNormal(fmt.Sprintf("initialize: time is %s (today=%s)", lastInitTime.Format("15:04:05"), today.Format("2006-01-02")))
	sunrise, sunset := SunriseSunset(lastInitTime, sunLat, sunLon)
	LogNormal(fmt.Sprintf("sunrise=%s sunset=%s", sunrise.Format("15:04"), sunset.Format("15:04")))

	newJobs := make([]*Job, 0, 20)

	newJobs = append(newJobs, NewJob("OfficeGarage+Street down", minTime(sunset.Add(10*time.Minute), today, 20), func() {
		Registry.OfficeGarage.MoveDown()
		Registry.OfficeStreet.MoveDown()
	}, false))

	newJobs = append(newJobs, NewJob("OfficeGarage+Street up", maxTime(sunrise.Add(10*time.Minute), today, 8.25), func() {
		Registry.OfficeGarage.MoveUp()
		Registry.OfficeStreet.MoveUp()
	}, false))

	newJobs = append(newJobs, NewJob("Kitchen+DiningRoom down", minTime(sunset.Add(8*time.Minute), today, 22), func() {
		Registry.Kitchen.MoveDown()
		Registry.DiningRoom.MoveDown()
		time.Sleep(1 * time.Second)
		Registry.Kitchen.MoveDown()
		Registry.DiningRoom.MoveDown()
	}, false))

	newJobs = append(newJobs, NewJob("Kitchen+DiningRoom up", maxTime(sunrise.Add(8*time.Minute), today, 7.45), func() {
		Registry.Kitchen.MoveUp()
		time.Sleep(1 * time.Second)
		Registry.DiningRoom.MoveUp()
		Registry.Kitchen.MoveUp()
		time.Sleep(1 * time.Second)
		Registry.DiningRoom.MoveUp()
	}, false))

	newJobs = append(newJobs, NewJob("SleepingRoom down", minTime(sunset, today, 22), func() {
		Registry.SleepingRoom.MoveDown()
	}, false))

	newJobs = append(newJobs, NewJob("PaulsRoom down", minTime(sunset.Add(2*time.Minute), today, 22), func() {
		Registry.PaulsRoom.MoveDown()
	}, false))

	newJobs = append(newJobs, NewJob("LeasRoom down", minTime(sunset.Add(1*time.Minute), today, 22), func() {
		Registry.LeasRoom.MoveDown()
	}, false))

	newJobs = append(newJobs, NewJob("LeasRoom up", today.Add(9*time.Hour), func() {
		Registry.LeasRoom.MoveUp()
	}, false))

	newJobs = append(newJobs, NewJob("RaffstoreLiving down", minTime(sunset.Add(4*time.Minute), today, 23), func() {
		Registry.RaffstoreLiving.MoveDown()
	}, false))

	newJobs = append(newJobs, NewJob("RaffstoreDining down", minTime(sunset.Add(3*time.Minute), today, 22), func() {
		Registry.RaffstoreDining.MoveDown()
	}, false))

	newJobs = append(newJobs, NewJob("RaffstoreDining+Living up", livingRoomRaffstoresUpTime(lastInitTime, today, sunrise), func() {
		Registry.RaffstoreDining.MoveUp()
		Registry.RaffstoreLiving.MoveUp()
	}, false))

	dailyHighTemp = nil

	isSummer := lastInitTime.Month() > 3 && lastInitTime.Month() < 10
	if isSummer {
		newJobs = append(newJobs, NewJob("OfficeStreet half", today.Add(13*time.Hour+30*time.Minute), func() {
			Registry.OfficeStreet.MoveHalf()
		}, false))

		newJobs = append(newJobs, NewJob("PaulsRoom+Leas 3/4", today.Add(9*time.Hour), func() {
			if dailyHighTemp == nil || *dailyHighTemp <= hotThresholdCelsius {
				LogNormal("[Job] PaulsRoom+Leas 3/4 skipped (temp not above threshold)")
				return
			}
			Registry.PaulsRoom.MoveThreeQuarters()
			time.Sleep(1 * time.Minute)
			Registry.LeasRoom.MoveThreeQuarters()
		}, false))

		newJobs = append(newJobs, NewJob("PaulsRoom+Leas up", today.Add(17*time.Hour+30*time.Minute), func() {
			if dailyHighTemp == nil || *dailyHighTemp <= hotThresholdCelsius {
				LogNormal("[Job] PaulsRoom+Leas up skipped (temp not above threshold)")
				return
			}
			Registry.PaulsRoom.MoveUp()
			time.Sleep(1 * time.Minute)
			Registry.LeasRoom.MoveUp()
		}, false))
	}

	jobsMu.Lock()
	jobs = newJobs
	jobsMu.Unlock()
}

func startHTTPServer() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("/mobilegate", func(w http.ResponseWriter, r *http.Request) {
		response := Registry.OfficeGarage.SendRequest(NewVersionRequest(), 2000)
		if strings.Contains(response, "VERSION_RES") {
			w.Write([]byte("ok"))
		} else {
			w.Write([]byte("down"))
		}
	})

	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(version))
	})

	mux.HandleFunc("/diagnostics", func(w http.ResponseWriter, r *http.Request) {
		type deviceEntry struct {
			Channel int `json:"channel"`
			Type    int `json:"type"`
		}
		type thingEntry struct {
			Channel int        `json:"channel"`
			Name    string     `json:"name"`
			Type    ThingType  `json:"type"`
			HwType  int        `json:"hwType"`
			State   *ThingState `json:"state"`
		}

		var devs []deviceEntry
		for i, dt := range deviceTypes {
			if dt != 0 {
				devs = append(devs, deviceEntry{Channel: i, Type: dt})
			}
		}

		var things []thingEntry
		for _, t := range Registry.All {
			hwType := -1
			if t.Channel < len(deviceTypes) {
				hwType = deviceTypes[t.Channel]
			}
			s, _ := StateCache.Get(t.Channel)
			things = append(things, thingEntry{
				Channel: t.Channel, Name: t.Name, Type: t.Type, HwType: hwType, State: s,
			})
		}

		schedulerStatus := "running"
		if schedulerWaitingNTP.Load() {
			schedulerStatus = "waiting_for_ntp_sync"
		}

		result := map[string]interface{}{
			"addonVersion":    version,
			"schedulerStatus": schedulerStatus,
			"firmware":     firmwareVersion,
			"hardware":     hardwareVersion,
			"enet":         enetVersion,
			"deviceTypes":  devs,
			"things":       things,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	mux.HandleFunc("/joblog", func(w http.ResponseWriter, r *http.Request) {
		cutoff := time.Now().AddDate(0, 0, -10).Format("2006-01-02")
		entries := FilterJobLog(GetNormalLog(), cutoff)
		w.Write([]byte(strings.Join(entries, "\n")))
	})

	mux.HandleFunc("/joblog/debug", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Join(GetDebugLog(), "\n")))
	})

	mux.HandleFunc("/things", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		type thingResponse struct {
			Channel int         `json:"channel"`
			Name    string      `json:"name"`
			Type    ThingType   `json:"type"`
			State   *ThingState `json:"state"`
		}
		var result []thingResponse
		for _, t := range Registry.All {
			s, _ := StateCache.Get(t.Channel)
			result = append(result, thingResponse{
				Channel: t.Channel, Name: t.Name, Type: t.Type, State: s,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	// Route: /things/{channel}/{action}[/{value}]
	mux.HandleFunc("/things/", func(w http.ResponseWriter, r *http.Request) {
		handleThingAction(w, r)
	})

	LogNormal("HTTP server starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		LogNormal(fmt.Sprintf("HTTP server error: %s", err.Error()))
	}
}

func handleThingAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse: /things/{channel}/{action}[/{value}]
	path := strings.TrimPrefix(r.URL.Path, "/things/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	channel, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "Invalid channel", http.StatusBadRequest)
		return
	}
	action := parts[1]

	var thing *Thing
	for _, t := range Registry.All {
		if t.Channel == channel {
			thing = t
			break
		}
	}
	if thing == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	switch action {
	case "up":
		switch thing.Type {
		case ThingTypeBlind:
			thing.MoveUp()
		case ThingTypeSwitch:
			thing.TurnOff()
		case ThingTypeDimmer:
			thing.DimmerTurnOff()
		default:
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
	case "down":
		switch thing.Type {
		case ThingTypeBlind:
			thing.MoveDown()
		case ThingTypeSwitch:
			thing.TurnOn()
		case ThingTypeDimmer:
			thing.DimmerTurnOn()
		default:
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
	case "position":
		if thing.Type != ThingTypeBlind {
			http.Error(w, "Not a blind", http.StatusBadRequest)
			return
		}
		if len(parts) < 3 {
			http.Error(w, "Missing value", http.StatusBadRequest)
			return
		}
		value, err := strconv.Atoi(parts[2])
		if err != nil {
			http.Error(w, "Invalid value", http.StatusBadRequest)
			return
		}
		thing.MoveTo(value)
	case "brightness":
		if thing.Type != ThingTypeDimmer {
			http.Error(w, "Not a dimmable light", http.StatusBadRequest)
			return
		}
		if len(parts) < 3 {
			http.Error(w, "Missing value", http.StatusBadRequest)
			return
		}
		value, err := strconv.Atoi(parts[2])
		if err != nil {
			http.Error(w, "Invalid value", http.StatusBadRequest)
			return
		}
		thing.SetBrightness(value)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	go refreshChannel(thing)
	w.WriteHeader(http.StatusOK)
}
