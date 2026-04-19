package main

import (
	"math"
	"time"
)

// SunPosition computes sunrise/sunset for the given location and date
// using a simplified astronomical algorithm (no external dependencies).

const (
	sunLat = 50.921210
	sunLon = 7.086539
)

func toRad(deg float64) float64 { return deg * math.Pi / 180 }
func toDeg(rad float64) float64 { return rad * 180 / math.Pi }

// SunriseSunset returns local sunrise and sunset times for the given date.
func SunriseSunset(date time.Time, lat, lon float64) (sunrise, sunset time.Time) {
	// Julian day
	y, m, d := date.Date()
	jd := julianDay(y, int(m), d)

	// Solar noon
	n := math.Floor(jd - 2451545.0 + 0.0008)
	jStar := n - lon/360.0
	M := math.Mod(357.5291+0.98560028*jStar, 360)
	C := 1.9148*math.Sin(toRad(M)) + 0.0200*math.Sin(toRad(2*M)) + 0.0003*math.Sin(toRad(3*M))
	lambda := math.Mod(M+C+180+102.9372, 360)
	jTransit := 2451545.0 + jStar + 0.0053*math.Sin(toRad(M)) - 0.0069*math.Sin(toRad(2*lambda))

	sinDec := math.Sin(toRad(lambda)) * math.Sin(toRad(23.4393))
	cosDec := math.Cos(math.Asin(sinDec))

	cosOmega := (math.Sin(toRad(-0.833)) - math.Sin(toRad(lat))*sinDec) / (math.Cos(toRad(lat)) * cosDec)

	// Clamp to [-1, 1] for polar regions
	if cosOmega > 1 {
		cosOmega = 1
	} else if cosOmega < -1 {
		cosOmega = -1
	}

	omega := toDeg(math.Acos(cosOmega))

	jRise := jTransit - omega/360.0
	jSet := jTransit + omega/360.0

	loc := date.Location()
	sunrise = julianToTime(jRise, loc)
	sunset = julianToTime(jSet, loc)
	return
}

func julianDay(year, month, day int) float64 {
	if month <= 2 {
		year--
		month += 12
	}
	A := year / 100
	B := 2 - A + A/4
	return math.Floor(365.25*float64(year+4716)) +
		math.Floor(30.6001*float64(month+1)) +
		float64(day) + float64(B) - 1524.5
}

func julianToTime(jd float64, loc *time.Location) time.Time {
	// Convert Julian date to Unix timestamp
	unixSeconds := (jd - 2440587.5) * 86400
	sec := int64(unixSeconds)
	nsec := int64((unixSeconds - float64(sec)) * 1e9)
	return time.Unix(sec, nsec).In(loc)
}
