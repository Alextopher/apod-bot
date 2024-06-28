package cache

// HasDate is an interface for types that have a date based timestamp
type HasDate interface {
	GetDate() string
}

// Cache interface for caching generic types.
type Cache[T any] interface {
	Add(string, T) error
	Get(string) (T, bool)
	Has(string) bool
}

// AddAll is a helper function to add a list of responses to a cache.
func AddAll[T HasDate](c Cache[T], responses []T) error {
	for _, response := range responses {
		if err := c.Add(response.GetDate(), response); err != nil {
			return err
		}
	}
	return nil
}
