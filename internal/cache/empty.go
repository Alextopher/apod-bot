package cache

// Empty cache that does nothing
type Empty[T any] struct{}

// NewEmptyCache creates a new EmptyCache
func NewEmptyCache[T any]() *Empty[T] {
	return &Empty[T]{}
}

// Add a single day to the cache
func (c *Empty[T]) Add(date string, response T) error {
	return nil
}

// Get a single day from the cache
func (c *Empty[T]) Get(date string) (T, bool) {
	var response T
	return response, false
}

// Has checks if a date is in the cache
func (c *Empty[T]) Has(date string) bool {
	return false
}
