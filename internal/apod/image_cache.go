package apod

import (
	"container/list"
	"os"
)

// ImageCache is a cache for images
//
// The cache only stores the image data (as a byte slice) along with the format
// of the image (e.g. "png", "jpg", etc.)
type ImageCache interface {
	// Get the image data for a date
	Get(date string) (wrapper *ImageWrapper, ok bool)
	// Set the image data for a date
	Set(date string, wrapper *ImageWrapper)
}

// GetOrSetImage gets the image data for a date from the cache, or sets it if
// it is not already in the cache
func GetOrSetImage(cache ImageCache, date string, getImage func() (wrapper *ImageWrapper, err error)) (wrapper *ImageWrapper, err error) {
	// Get the image from the cache
	wrapper, ok := cache.Get(date)
	if ok {
		return wrapper, nil
	}

	// Get the image from the source
	wrapper, err = getImage()
	if err != nil {
		return wrapper, err
	}

	// Set the image in the cache
	cache.Set(date, wrapper)

	return wrapper, nil
}

// MemoryImageCache is an in-memory image cache
//
// It is a basic LRU cache with a maximum size of ~100MB
type MemoryImageCache struct {
	MaxSize int
	Size    int
	Cache   list.List
}

// MemoryImageCacheEntry is an entry in the MemoryImageCache
type memoryImageCacheEntry struct {
	Date    string
	Wrapper *ImageWrapper
}

// NewMemoryImageCache creates a new MemoryImageCache
func NewMemoryImageCache(maxSize int) *MemoryImageCache {
	return &MemoryImageCache{
		MaxSize: maxSize,
	}
}

// Get the image data for a date
func (c *MemoryImageCache) Get(date string) (wrapper *ImageWrapper, ok bool) {
	for e := c.Cache.Front(); e != nil; e = e.Next() {
		if e.Value.(*memoryImageCacheEntry).Date == date {
			c.Cache.MoveToFront(e)
			return e.Value.(*memoryImageCacheEntry).Wrapper, true
		}
	}
	return nil, false
}

// Set the image data for a date
func (c *MemoryImageCache) Set(date string, wrapper *ImageWrapper) {
	// If the image is already in the cache, update it
	for e := c.Cache.Front(); e != nil; e = e.Next() {
		if e.Value.(*memoryImageCacheEntry).Date == date {
			c.Cache.MoveToFront(e)
			e.Value.(*memoryImageCacheEntry).Wrapper = wrapper
			return
		}
	}

	// If the cache is full, remove the oldest entries
	if c.Size >= c.MaxSize {
		e := c.Cache.Remove(c.Cache.Back()).(*memoryImageCacheEntry)
		c.Size -= len(e.Wrapper.Bytes)
	}
}

// DirectoryImageCache is an image cache that stores images in a directory
//
// It has unlimited size
type DirectoryImageCache struct {
	Directory string
}

// NewDirectoryImageCache creates a new DirectoryImageCache
//
// Attempts to create the directory if it does not exist
func NewDirectoryImageCache(directory string) (*DirectoryImageCache, error) {
	err := os.MkdirAll(directory, 0755)
	if err != nil {
		return nil, err
	}

	return &DirectoryImageCache{
		Directory: directory,
	}, nil
}

// Get the image data for a date
func (c *DirectoryImageCache) Get(date string) (wrapper *ImageWrapper, ok bool) {
	// Images are stored in the format <date>.<format>
	formats := []string{"png", "jpg", "jpeg", "gif"}
	for _, format := range formats {
		data, err := os.ReadFile(c.Directory + "/" + date + "." + format)
		if err != nil {
			continue
		}

		wrapper, err := NewImageWrapper(data)
		if err != nil {
			continue
		}

		return wrapper, true
	}

	return nil, false
}

// Set the image data for a date
func (c *DirectoryImageCache) Set(date string, wrapper *ImageWrapper) {
	os.WriteFile(c.Directory+"/"+date+"."+wrapper.Format, wrapper.Bytes, 0644)
}

// NullImageCache is an image cache that does not store anything
type NullImageCache struct{}

// NewNullImageCache creates a new NullImageCache
func NewNullImageCache() *NullImageCache {
	return &NullImageCache{}
}

// Get the image data for a date
func (c *NullImageCache) Get(date string) (wrapper *ImageWrapper, ok bool) {
	return nil, false
}

// Set the image data for a date
func (c *NullImageCache) Set(date string, wrapper *ImageWrapper) {}
