package cache

import (
	"encoding/json"
	"io"
	"sync"
)

// AppendOnly is a thread-safe cache that allows appending new items.
type AppendOnly[T HasDate] struct {
	sync.RWMutex
	cache   map[string]T
	encoder *json.Encoder
}

// NewAppendCache creates a new APODCache
func NewAppendCache[T HasDate](r io.Reader, w io.Writer) (*AppendOnly[T], error) {
	cache := &AppendOnly[T]{
		cache:   make(map[string]T),
		encoder: json.NewEncoder(w),
	}
	if err := cache.load(r); err != nil {
		return nil, err
	}
	return cache, nil
}

// Load days from a reader
func (c *AppendOnly[T]) load(r io.Reader) error {
	dec := json.NewDecoder(r)
	for {
		var response T
		if err := dec.Decode(&response); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		c.cache[response.GetDate()] = response
	}
	return nil
}

// Add a single day to the cache
func (c *AppendOnly[T]) Add(date string, response T) error {
	c.Lock()
	c.cache[date] = response
	err := c.encoder.Encode(response)
	c.Unlock()
	return err
}

// Get a single day from the cache
func (c *AppendOnly[T]) Get(date string) (T, bool) {
	c.RLock()
	response, ok := c.cache[date]
	c.RUnlock()
	return response, ok
}

// Has checks if a day is present in the cache
func (c *AppendOnly[T]) Has(date string) bool {
	c.RLock()
	_, ok := c.cache[date]
	c.RUnlock()
	return ok
}
