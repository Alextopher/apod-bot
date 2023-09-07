package main

import (
	"encoding/json"
	"io"
	"sync"
)

// Saves APOD json responses to an append-only file and keeps a map of the current state
type APODCache struct {
	sync.RWMutex
	encoder *json.Encoder
	cache   map[string]APODResponse
}

// Helper function to create a new cache
func NewAPODCache(r io.Reader, w io.Writer) (*APODCache, error) {
	cache := &APODCache{
		cache:   make(map[string]APODResponse),
		encoder: json.NewEncoder(w),
	}
	if err := cache.Load(r); err != nil {
		return nil, err
	}
	return cache, nil
}

// Load days from a reader
func (c *APODCache) Load(r io.Reader) error {
	dec := json.NewDecoder(r)
	for {
		var response APODResponse
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

// Adds a single day to the cache
func (c *APODCache) Add(response APODResponse) error {
	c.Lock()
	c.cache[response.Date] = response
	err := c.encoder.Encode(response)
	c.Unlock()
	return err
}

// Adds multiple days to the cache
func (c *APODCache) AddAll(responses []APODResponse) error {
	c.Lock()
	defer c.Unlock()

	for _, response := range responses {
		c.cache[response.Date] = response
		if err := c.encoder.Encode(response); err != nil {
			return err
		}
	}
	return nil
}

// Writes the entire cache to a writer
func (c *APODCache) WriteAll(w io.Writer) error {
	c.RLock()
	defer c.RUnlock()

	encoder := json.NewEncoder(w)
	for _, response := range c.cache {
		if err := encoder.Encode(response); err != nil {
			return err
		}
	}
	return nil
}

// Gets a single day from the cache
func (c *APODCache) Get(date string) (APODResponse, bool) {
	c.RLock()
	response, ok := c.cache[date]
	c.RUnlock()
	return response, ok
}

// Cache's today's APOD image so we don't have to make multiple downloads for the same image
// Eventually, we'll want to expand this to save all days to disk
type ImageCache struct {
	sync.RWMutex
	day   string
	image []byte
}

func NewImageCache() *ImageCache {
	return &ImageCache{}
}

// Saves the image to the cache
func (c *ImageCache) Set(day string, image []byte) {
	c.Lock()
	c.day = day
	c.image = image
	c.Unlock()
}

// Gets the image from the cache
func (c *ImageCache) Get(day string) ([]byte, bool) {
	c.RLock()
	defer c.RUnlock()

	if c.day == day {
		return c.image, true
	}
	return nil, false
}

// Gets the image from the cache or sets it using the provided function
func (c *ImageCache) GetOrSet(day string, fn func() ([]byte, error)) ([]byte, error) {
	c.RLock()
	if c.day == day {
		c.RUnlock()
		return c.image, nil
	}
	c.RUnlock()

	c.Lock()
	image, err := fn()
	if err != nil {
		return nil, err
	}
	c.day = day
	c.image = image
	c.Unlock()

	return image, nil
}
