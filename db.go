package main

import (
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"
)

type DB struct {
	sync.RWMutex
	encoder *json.Encoder
	// maps channelID to the hour (utc) to send the APOD message
	schedule map[string]int
	// maps channelID to the date of the last APOD sent
	last map[string]string
}

type EventType int

const (
	EventTypeSet EventType = iota
	EventTypeRemove
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

// Define marshalling/unmarshalling for EventType
func (e EventType) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

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

type Event struct {
	Time time.Time `json:"time"`
	Type EventType `json:"type"`
	// /schedule
	Set *SetEvent `json:"set,omitempty"`
	// /stop
	Remove *RemoveEvent `json:"remove,omitempty"`
	// tracks the last APOD sent to this channel
	Sent *SentEvent `json:"sent,omitempty"`
}

type SetEvent struct {
	ChannelID string `json:"channel_id"`
	Hour      int    `json:"hour"`
}

type RemoveEvent struct {
	ChannelID string `json:"channel_id"`
}

type SentEvent struct {
	ChannelID string `json:"channel_id"`
	Date      string `json:"date"`
}

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

// Removes entries that match a predicate
func (db *DB) RemoveIf(f func(string, int) bool) {
	db.Lock()
	for channelID, hour := range db.schedule {
		if f(channelID, hour) {
			delete(db.schedule, channelID)
		}
	}
	db.Unlock()
}

// Iterates over all entries
func (db *DB) View(f func(string, int)) {
	db.RLock()
	for channelID, hour := range db.schedule {
		f(channelID, hour)
	}
	db.RUnlock()
}

// Returns the number of entries
func (db *DB) Size() int {
	db.RLock()
	length := len(db.schedule)
	db.RUnlock()
	return length
}

// Returns the date of the last APOD sent to a channel
func (db *DB) GetLast(channelID string) (string, bool) {
	db.RLock()
	date, ok := db.last[channelID]
	db.RUnlock()
	return date, ok
}
