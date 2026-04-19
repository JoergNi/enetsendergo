package main

import "sync"

// ThingRegistry holds all known eNet devices and their cached states.
var Registry = struct {
	Schrank          *Thing
	OfficeStreet     *Thing
	OfficeGarage     *Thing
	RaffstoreDining  *Thing
	RaffstoreLiving  *Thing
	DiningRoom       *Thing
	SleepingRoom     *Thing
	Kitchen          *Thing
	LeasRoom         *Thing
	PaulsRoom        *Thing
	LivingEsszimmer  *Thing
	LivingWohnbereich *Thing
	All              []*Thing
}{}

// StateCache is a thread-safe map from channel to ThingState.
type stateCache struct {
	mu    sync.RWMutex
	cache map[int]*ThingState
}

var StateCache = &stateCache{cache: make(map[int]*ThingState)}

func (c *stateCache) Get(channel int) (*ThingState, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s, ok := c.cache[channel]
	return s, ok
}

func (c *stateCache) Set(channel int, state *ThingState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[channel] = state
}

func InitRegistry(sender MobilegateSender) {
	s := sender
	Registry.Schrank          = &Thing{Name: "Schrank", Channel: 16, Type: ThingTypeSwitch, Sender: s}
	Registry.OfficeStreet     = &Thing{Name: "RolloArbeitszimmerStraße", Channel: 17, Type: ThingTypeBlind, Sender: s}
	Registry.OfficeGarage     = &Thing{Name: "RolloArbeitszimmerGarage", Channel: 18, Type: ThingTypeBlind, Sender: s}
	Registry.RaffstoreDining  = &Thing{Name: "RaffstoreEssen", Channel: 19, Type: ThingTypeBlind, Sender: s}
	Registry.RaffstoreLiving  = &Thing{Name: "RaffstoreTerassenTür", Channel: 20, Type: ThingTypeBlind, Sender: s}
	Registry.DiningRoom       = &Thing{Name: "RolloEssen", Channel: 21, Type: ThingTypeBlind, Sender: s}
	Registry.SleepingRoom     = &Thing{Name: "RolloSchlafzimmer", Channel: 22, Type: ThingTypeBlind, Sender: s}
	Registry.Kitchen          = &Thing{Name: "RolloKueche", Channel: 23, Type: ThingTypeBlind, Sender: s}
	Registry.LeasRoom         = &Thing{Name: "RolloLeasZimmer", Channel: 24, Type: ThingTypeBlind, Sender: s}
	Registry.PaulsRoom        = &Thing{Name: "RolloPaulsZimmer", Channel: 25, Type: ThingTypeBlind, Sender: s}
	Registry.LivingEsszimmer  = &Thing{Name: "LichtEsszimmer", Channel: 27, Type: ThingTypeDimmer, Sender: s}
	Registry.LivingWohnbereich = &Thing{Name: "LichtWohnbereich", Channel: 28, Type: ThingTypeDimmer, Sender: s}

	Registry.All = []*Thing{
		Registry.Schrank, Registry.OfficeStreet, Registry.OfficeGarage,
		Registry.RaffstoreDining, Registry.RaffstoreLiving,
		Registry.DiningRoom, Registry.SleepingRoom, Registry.Kitchen,
		Registry.LeasRoom, Registry.PaulsRoom,
		Registry.LivingWohnbereich, Registry.LivingEsszimmer,
	}
}
