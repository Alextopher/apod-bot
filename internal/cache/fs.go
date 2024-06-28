package cache

import "os"

// FSCache is a cache that stores cached items in a "file system".
type FSCache[T any] struct {
	fs           FS
	serializer   func([]byte) (T, error)
	deserializer func(T) ([]byte, error)
}

// NewFSCache creates a new FSCache.
func NewFSCache[T any](
	fs FS,
	serializer func([]byte) (T, error),
	deserializer func(T) ([]byte, error),
) *FSCache[T] {
	return &FSCache[T]{
		fs, serializer, deserializer,
	}
}

// Add a single day to the cache.
func (c *FSCache[T]) Add(date string, response T) error {
	bytes, err := c.deserializer(response)
	if err != nil {
		return err
	}
	return c.fs.WriteFile(date, bytes)
}

// Get a single day from the cache.
func (c *FSCache[T]) Get(date string) (T, bool) {
	var response T
	data, err := c.fs.ReadFile(date)
	if err != nil {
		return response, false
	}

	response, err = c.serializer(data)
	if err != nil {
		return response, false
	}

	return response, true
}

// Has checks if a date is in the cache.
func (c *FSCache[T]) Has(date string) bool {
	return c.fs.HasFile(date)
}

// FS is an interface that abstracts over types that behave like a file system.
// This includes both a local file system and a mock in-memory file system.
type FS interface {
	HasFile(name string) bool
	WriteFile(name string, data []byte) error
	ReadFile(name string) ([]byte, error)
}

// LocalFS is a file system that interacts with the local file system through a base directory.
type LocalFS struct {
	baseDir string
}

// NewLocalFS creates a new LocalFS.
func NewLocalFS(baseDir string) *LocalFS {
	return &LocalFS{baseDir}
}

// HasFile checks if a file exists.
func (fs *LocalFS) HasFile(name string) bool {
	_, err := os.Stat(fs.baseDir + "/" + name)
	return err == nil
}

// WriteFile writes data to a file.
func (fs *LocalFS) WriteFile(name string, data []byte) error {
	return os.WriteFile(fs.baseDir+"/"+name, data, 0644)
}

// ReadFile reads data from a file.
func (fs *LocalFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(fs.baseDir + "/" + name)
}

// InMemoryFS is a file system that stores files in memory.
type InMemoryFS struct {
	files map[string][]byte
}

// NewInMemoryFS creates a new InMemoryFS.
func NewInMemoryFS() *InMemoryFS {
	return &InMemoryFS{make(map[string][]byte)}
}

// HasFile checks if a file exists.
func (fs *InMemoryFS) HasFile(name string) bool {
	_, ok := fs.files[name]
	return ok
}

// WriteFile writes data to a file.
func (fs *InMemoryFS) WriteFile(name string, data []byte) error {
	fs.files[name] = data
	return nil
}

// ReadFile reads data from a file.
func (fs *InMemoryFS) ReadFile(name string) ([]byte, error) {
	data, ok := fs.files[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return data, nil
}
