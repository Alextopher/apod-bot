package main

import (
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"
)

// DB is the bot's database
type DB struct {
	sync.RWMutex
	encoder *json.Encoder
	// maps channelID to the hour (utc) to send the APOD message
	schedule map[string]int
	// maps channelID to the date of the last APOD sent
	last map[string]string
}

// EventType enum
type EventType int

const (
	// EventTypeSet is a set event (/schedule)
	EventTypeSet EventType = iota
	// EventTypeRemove is a remove event (/stop)
	EventTypeRemove
	// EventTypeSent is a sent event (APOD sent)
	EventTypeSent
)

func (e EventType) String() string {
	switch e {
	case EventTypeSet:
		return "set"
	case EventTypeRemove:
		return "remove"
	case EventTypeSent:
		return "sent"
	}

	return ""
}

// MarshalJSON for EventType
func (e EventType) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

// UnmarshalJSON for EventType
func (e *EventType) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	switch s {
	case "set":
		*e = EventTypeSet
	case "remove":
		*e = EventTypeRemove
	case "sent":
		*e = EventTypeSent
	default:
		return errors.New("invalid event type")
	}

	return nil
}

// Event is a database event
type Event struct {
	// Time is the time the event occurred
	Time time.Time `json:"time"`
	// Type is the type of event
	Type EventType `json:"type"`
	// Set is the set event (/schedule)
	Set *SetEvent `json:"set,omitempty"`
	// Remove is the remove event (/stop)
	Remove *RemoveEvent `json:"remove,omitempty"`
	// Sent is the sent event (APOD sent)
	Sent *SentEvent `json:"sent,omitempty"`
}

// SetEvent adds a channel to the schedule
type SetEvent struct {
	// ChannelID is the discord channel ID
	ChannelID string `json:"channel_id"`
	// Hour is the hour (utc) to send the APOD message
	Hour int `json:"hour"`
}

// RemoveEvent removes a channel from the schedule
type RemoveEvent struct {
	// ChannelID is the discord channel ID
	ChannelID string `json:"channel_id"`
}

// SentEvent tracks the last APOD sent to a channel
type SentEvent struct {
	// ChannelID is the discord channel ID
	ChannelID string `json:"channel_id"`
	// Date is the date of the last APOD sent
	Date string `json:"date"`
}

// NewDB creates a new DB
func NewDB(r io.Reader, w io.Writer) (*DB, error) {
	db := &DB{
		encoder:  json.NewEncoder(w),
		schedule: make(map[string]int),
		last:     make(map[string]string),
	}
	if err := db.load(r); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *DB) load(r io.Reader) error {
	dec := json.NewDecoder(r)
	for {
		var event Event
		if err := dec.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		switch event.Type {
		case EventTypeSet:
			db.set(event.Set)
		case EventTypeRemove:
			db.remove(event.Remove)
		case EventTypeSent:
			db.sent(event.Sent)
		}
	}

	return nil
}

func (db *DB) set(event *SetEvent) {
	db.schedule[event.ChannelID] = event.Hour
}

func (db *DB) remove(event *RemoveEvent) {
	delete(db.schedule, event.ChannelID)
}

func (db *DB) sent(event *SentEvent) {
	db.last[event.ChannelID] = event.Date
}

// Set adds a channel to the schedule
func (db *DB) Set(channelID string, hour int) {
	db.Lock()
	event := &Event{
		Time: time.Now(),
		Type: EventTypeSet,
		Set: &SetEvent{
			ChannelID: channelID,
			Hour:      hour,
		},
	}
	db.set(event.Set)
	db.encoder.Encode(event)
	db.Unlock()
}

// Remove removes a channel from the schedule
func (db *DB) Remove(channelID string) {
	db.Lock()
	event := &Event{
		Time: time.Now(),
		Type: EventTypeRemove,
		Remove: &RemoveEvent{
			ChannelID: channelID,
		},
	}
	db.remove(event.Remove)
	db.encoder.Encode(event)
	db.Unlock()
}

// Sent tracks the last APOD sent to a channel
func (db *DB) Sent(channelID, date string) {
	db.Lock()
	event := &Event{
		Time: time.Now(),
		Type: EventTypeSent,
		Sent: &SentEvent{
			ChannelID: channelID,
			Date:      date,
		},
	}
	db.sent(event.Sent)
	db.encoder.Encode(event)
	db.Unlock()
}

// RemoveIf removes all entries that match the given predicate
func (db *DB) RemoveIf(f func(string, int) bool) {
	db.Lock()
	for channelID, hour := range db.schedule {
		if f(channelID, hour) {
			event := &Event{
				Time: time.Now(),
				Type: EventTypeRemove,
				Remove: &RemoveEvent{
					ChannelID: channelID,
				},
			}
			db.remove(event.Remove)
			db.encoder.Encode(event)
		}
	}
	db.Unlock()
}

// View iterates over all entries in the database
func (db *DB) View(f func(string, int)) {
	db.RLock()
	for channelID, hour := range db.schedule {
		f(channelID, hour)
	}
	db.RUnlock()
}

// Size returns the number of entries in the database
func (db *DB) Size() int {
	db.RLock()
	length := len(db.schedule)
	db.RUnlock()
	return length
}

// GetLast returns the date of the last APOD sent to a channel
func (db *DB) GetLast(channelID string) (string, bool) {
	db.RLock()
	date, ok := db.last[channelID]
	db.RUnlock()
	return date, ok
}
