package apod

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/Alextopher/apod-bot/internal/cache"
)

// APOD is a client for the NASA APOD API
//
// It maintains a cache for APOD responses and an image cache for images.
type APOD struct {
	key        string
	cache      cache.Cache[*Response]
	imageCache cache.Cache[*ImageWrapper]
	// To avoid issues with timezones we keep track of the most recent APOD response date
	// and we only update that date at most once per hour
	lastUpdate time.Time
	current    *Response
}

// NewClient creates a new APOD client
func NewClient(key string, cache cache.Cache[*Response], imageCache cache.Cache[*ImageWrapper]) *APOD {
	return &APOD{
		key:        key,
		cache:      cache,
		imageCache: imageCache,
		lastUpdate: time.Unix(0, 0), // the past
		current:    nil,
	}
}

// IsValidDate checks if a date is formatted correctly and occurs after the first published APOD
func IsValidDate(date string) bool {
	// Check if the date is in the correct format
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false
	}

	start := time.Date(1995, 6, 16, 0, 0, 0, 0, time.UTC)
	return d.After(start)
}

var (
	// ErrorDateNotFound is returned when given date is not found on the NASA API
	ErrorDateNotFound = fmt.Errorf("date not found in NASA API")
	// ErrorDateInvalid is returned when given date is not in the correct format
	ErrorDateInvalid = fmt.Errorf("date is not in the correct format, use yyyy-mm-dd")
)

// singleRequest makes an HTTP request to the NASA API and expects a single APOD response
func (a *APOD) singleRequest(req string) (*Response, error) {
	resp, err := http.Get(req)
	if err != nil {
		return nil, err
	}

	// Check for non-200 status code
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrorDateNotFound
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NASA API Failure: %s", resp.Status)
	}

	// Decode the JSON response
	var response Response
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// rangeRequest returns all APODs between two dates (inclusive)
func (a *APOD) rangeRequest(start, end string) ([]*Response, error) {
	log.Println("Getting APODs from", start, "to", end)

	// Get the JSON response from the API
	req := fmt.Sprintf("https://api.nasa.gov/planetary/apod?thumbs=true&start_date=%s&end_date=%s&api_key=%s", start, end, a.key)
	resp, err := http.Get(req)
	if err != nil {
		return nil, err
	}

	// Check for non-200 status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NASA API Failure: %s", resp.Status)
	}

	// Decode the JSON response
	var responses []*Response
	err = json.NewDecoder(resp.Body).Decode(&responses)
	if err != nil {
		return nil, err
	}

	// Add the responses to the cache
	err = cache.AddAll(a.cache, responses)
	return responses, err
}

// Get the APOD response for a specific date
//
// Uses the cache if the response is already stored
func (a *APOD) Get(date string) (response *Response, err error) {
	// Check if the date is valid and if not, send an error message.
	if !IsValidDate(date) {
		return nil, ErrorDateInvalid
	}

	// If the cache has the response, return that
	if resp, ok := a.cache.Get(date); ok {
		return resp, err
	}

	req := fmt.Sprintf("https://api.nasa.gov/planetary/apod?thumbs=true&date=%s&api_key=%s", date, a.key)
	response, err = a.singleRequest(req)

	if err != nil {
		return response, err
	}

	// Add the response to the cache
	a.cache.Add(response.Date, response)
	return response, nil
}

// Returns the APOD response for today
func (a *APOD) Today() (response *Response, err error) {
	if a.current != nil && time.Since(a.lastUpdate) < time.Hour {
		return a.current, nil
	}

	req := fmt.Sprintf("https://api.nasa.gov/planetary/apod?thumbs=true&api_key=%s", a.key)
	response, err = a.singleRequest(req)

	if err != nil {
		return response, err
	}

	// Add the response to the cache
	a.current = response
	a.lastUpdate = time.Now()
	a.cache.Add(response.Date, response)
	return response, nil
}

// Fill runs in the background and fills the cache with _ALL_ APOD responses from the NASA API
func (a *APOD) Fill() {
	// Must be after 1995-06-16 (first APOD)
	start := time.Date(1995, 6, 16, 0, 0, 0, 0, time.UTC)

	// Get today's date from APOD's perspective
	todayResp, err := a.Today()
	if err != nil {
		log.Println("Error getting today's APOD, so we can't start filling the cache:", err)
		return
	}
	today, _ := time.Parse("2006-01-02", todayResp.Date)

	// "Iterator" over every day from start to today
	for d := start; d.Before(today); d = d.AddDate(0, 0, 1) {
		// Skip days that are already cached
		if a.cache.Has(d.Format("2006-01-02")) {
			continue
		}

		// We found a gap in the cache, search to find the end of the gap
		end := d.AddDate(0, 0, 1)
		for !a.cache.Has(end.Format("2006-01-02")) && end.Sub(d) < 30*24*time.Hour && end.Before(today) {
			end = end.AddDate(0, 0, 1)
		}

		// Get the APODs for the gap
		_, err := a.rangeRequest(d.Format("2006-01-02"), end.Format("2006-01-02"))
		if err != nil {
			log.Println("Error getting APODs:", err)
			continue
		}

		d = end
		time.Sleep(5 * time.Second)
	}

	// Go back through, getting individual APODs that are still missing
	for d := start; d.Before(today); d = d.AddDate(0, 0, 1) {
		// Skip days that are already cached
		if a.cache.Has(d.Format("2006-01-02")) {
			continue
		}

		// Get the APOD for the day
		_, err := a.Get(d.Format("2006-01-02"))
		if err != nil {
			log.Println("Failed to get APOD for", d.Format("2006-01-02"), "on retry. This is probably fine:", err)
			continue
		}
		time.Sleep(1 * time.Second)
	}

	log.Println("Finished filling cache!!")
}

func (a *APOD) Random() (*Response, error) {
	// Must be after 1995-06-16 (first APOD) and before today
	start := time.Date(1995, 6, 16, 0, 0, 0, 0, time.UTC)
	current, err := a.Today()
	if err != nil {
		return nil, err
	}
	end, _ := time.Parse("2006-01-02", current.Date)

	// Get a random date between start and end
	diff := end.Sub(start)
	random := time.Duration(rand.Int63n(int64(diff)))
	date := start.Add(random).Format("2006-01-02")

	// Get the APOD for the random date (if the )
	return a.Get(date)
}

// GetImage returns the image for a specific day
func (a *APOD) GetImage(day string) (*ImageWrapper, error) {
	// Check if the date is valid and if not, send an error message.
	if !IsValidDate(day) {
		return nil, ErrorDateInvalid
	}

	// If the image cache has the image, return that
	if img, ok := a.imageCache.Get(day); ok {
		return img, nil
	}

	// Otherwise, we need to get the APOD response
	response, err := a.Get(day)
	if err != nil {
		return nil, err
	}

	// Get the image from the response
	image, err := response.DownloadRawImage()
	if err != nil {
		return nil, err
	}

	// Add the full size image to the cache
	a.imageCache.Add(day, image)
	return image, nil
}

// Retry is a helper function that reruns a function until it succeeds, at most 5 times
func Retry[T any](f func() (T, error)) (res T, err error) {
	// exponential backoff
	backOff := 64 * time.Millisecond

	for i := 0; i < 5; i++ {
		res, err = f()
		if err == nil {
			return res, nil
		}
		backOff *= 2
	}
	return res, err
}
