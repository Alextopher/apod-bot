package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
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

// ImageCache is an interface for caching images by day.
type ImageCache interface {
	Get(day string) ([]byte, string, bool)
	Set(day string, format string, image []byte)
}

// GetOrSet retrieves an image from the cache if it exists, otherwise it calls the provided function to get the image and stores it in the cache.
// It returns the image and an error if any.
func GetOrSet(cache ImageCache, day string, fn func() ([]byte, string, error)) (image []byte, format string, err error) {
	var ok bool
	image, format, ok = cache.Get(day)
	if !ok {
		image, format, err = fn()
		if err != nil {
			return nil, format, err
		}
		cache.Set(day, format, image)
	}
	return image, format, nil
}

// DirectoryImageCache is an ImageCache that stores images in a directory.
type DirectoryImageCache struct {
	sync.RWMutex
	dir string

	// if a day is in this map, then it is ready for use
	ready map[string]struct{}
}

// NewDirectoryImageCache creates a new DirectoryImageCache.
func NewDirectoryImageCache(dir string) (*DirectoryImageCache, error) {
	// Create the directory if it doesn't exist
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.Mkdir(dir, 0755)
		if err != nil {
			return nil, err
		}
	}

	cache := &DirectoryImageCache{
		dir:   dir,
		ready: make(map[string]struct{}),
	}

	// Populate the ready map
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		day := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		cache.ready[day] = struct{}{}
	}

	log.Println("Populated image cache with", len(cache.ready), "images")
	return cache, nil
}

// Get retrieves an image from the cache if it exists.
func (c *DirectoryImageCache) Get(day string) ([]byte, string, bool) {
	c.RLock()
	if _, ok := c.ready[day]; !ok {
		c.RUnlock()
		return nil, "", false
	}
	c.RUnlock()

	var err error
	var image []byte
	var format string

	for _, format = range []string{"jpg", "jpeg", "png", "gif"} {
		path := fmt.Sprintf("%s.%s", filepath.Join(c.dir, day), format)

		image, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if err != nil {
		log.Println("Error reading image for day", day, ":", err)
		return nil, "", false
	}

	return image, format, true
}

// Set stores an image in the cache.
func (c *DirectoryImageCache) Set(day string, format string, image []byte) {
	path := fmt.Sprintf("%s.%s", filepath.Join(c.dir, day), format)
	err := os.WriteFile(path, image, 0644)
	if err != nil {
		log.Println("Error writing image for day", day, ":", err)
		return
	} else {
		log.Println("Saved image for day", day)
	}

	c.Lock()
	c.ready[day] = struct{}{}
	c.Unlock()
}

type DiscardImageCache struct{}

func NewDiscardImageCache() *DiscardImageCache {
	return &DiscardImageCache{}
}

func (c *DiscardImageCache) Get(day string) ([]byte, string, bool) {
	return nil, "", false
}

func (c *DiscardImageCache) Set(day string, format string, image []byte) {}
