package apod

import (
	"encoding/json"
	"io"
	"sync"
)

// Cache is a cache for APOD responses
type Cache interface {
	// Load loads data from a reader
	Load(r io.Reader) error
	// Add adds a single response to the cache
	Add(response *Response) error
	// AddAll adds multiple responses to the cache
	AddAll(responses []*Response) error
	// WriteAll writes the entire cache to a writer
	WriteAll(w io.Writer) error
	// Get gets a single response from the cache
	Get(date string) (*Response, bool)
	// Has checks if a response is in the cache
	Has(date string) bool
	// Size returns the number of responses in the cache
	Size() int
}

// AppendOnly is an unconventional cache that only appends to a file
type AppendOnly struct {
	sync.RWMutex
	cache   map[string]*Response
	encoder *json.Encoder
}

// NewAPODCache creates a new APODCache
func NewAPODCache(r io.Reader, w io.Writer) (*AppendOnly, error) {
	cache := &AppendOnly{
		cache:   make(map[string]*Response),
		encoder: json.NewEncoder(w),
	}
	if err := cache.Load(r); err != nil {
		return nil, err
	}
	return cache, nil
}

// Load days from a reader
func (c *AppendOnly) Load(r io.Reader) error {
	dec := json.NewDecoder(r)
	for {
		var response *Response
		if err := dec.Decode(&response); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		c.cache[response.Date] = response
	}
	return nil
}

// Add a single day to the cache
func (c *AppendOnly) Add(response *Response) error {
	c.Lock()
	c.cache[response.Date] = response
	err := c.encoder.Encode(response)
	c.Unlock()
	return err
}

// AddAll adds multiple days to the cache
func (c *AppendOnly) AddAll(responses []*Response) error {
	c.Lock()
	for _, response := range responses {
		// If the cache already has the response, skip it
		if _, ok := c.cache[response.Date]; ok {
			continue
		}

		c.cache[response.Date] = response
		if err := c.encoder.Encode(response); err != nil {
			c.Unlock()
			return err
		}
	}
	c.Unlock()
	return nil
}

// WriteAll writes out the entire cache to a writer
func (c *AppendOnly) WriteAll(w io.Writer) error {
	c.RLock()
	encoder := json.NewEncoder(w)
	for _, response := range c.cache {
		if err := encoder.Encode(response); err != nil {
			c.RUnlock()
			return err
		}
	}
	c.RUnlock()
	return nil
}

// Get a single day from the cache
func (c *AppendOnly) Get(date string) (*Response, bool) {
	c.RLock()
	response, ok := c.cache[date]
	c.RUnlock()
	return response, ok
}

// Has checks if a day is in the cache
func (c *AppendOnly) Has(date string) bool {
	c.RLock()
	_, ok := c.cache[date]
	c.RUnlock()
	return ok
}

// Size returns the number of days in the cache
func (c *AppendOnly) Size() int {
	c.RLock()
	size := len(c.cache)
	c.RUnlock()
	return size
}
