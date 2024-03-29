package apod

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// APOD is a client for the NASA APOD API
//
// It has a cache for APOD responses and an image cache for images.
type APOD struct {
	key        string
	cache      Cache
	ImageCache ImageCache
}

// NewClient creates a new APOD client
func NewClient(key string, cache Cache, imageCache ImageCache) *APOD {
	return &APOD{
		key:        key,
		cache:      cache,
		ImageCache: imageCache,
	}
}

var (
	// ErrorDateNotFound is returned when the date is not found
	ErrorDateNotFound = fmt.Errorf("date not found")
)

// Get the APOD response for a specific date
func (a *APOD) Get(date string) (response *Response, err error) {
	// If the cache has the response, return that
	if resp, ok := a.cache.Get(date); ok {
		return resp, err
	}

	// Get the JSON response from the API
	req := fmt.Sprintf("https://api.nasa.gov/planetary/apod?thumbs=true&date=%s&api_key=%s", date, a.key)
	resp, err := http.Get(req)
	if err != nil {
		return response, err
	}

	// Check for non-200 status code
	if resp.StatusCode == http.StatusNotFound {
		return response, ErrorDateNotFound
	} else if resp.StatusCode != http.StatusOK {
		return response, fmt.Errorf("NASA API Failure: %s", resp.Status)
	}

	// Decode the JSON response
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return response, err
	}

	// Add the response to the cache
	a.cache.Add(response)
	return response, nil
}

// Range gets all APODs between two dates (inclusive)
func (a *APOD) Range(start, end string) ([]*Response, error) {
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

	log.Println("Got", len(responses), "APODs")

	// Add the responses to the cache
	err = a.cache.AddAll(responses)
	return responses, err
}

// Fill will fill the cache with _all_ APODs from the NASA API.
func (a *APOD) Fill() {
	// Must be after 1995-06-16 (first APOD)
	start := time.Date(1995, 6, 16, 0, 0, 0, 0, time.UTC)
	// Today's date
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

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
		_, err := a.Range(d.Format("2006-01-02"), end.Format("2006-01-02"))
		if err != nil {
			log.Println("Error getting APODs:", err)
			continue
		}

		d = end
		time.Sleep(5 * time.Second)
	}

	// Go back through, getting individual APODs that are missing
	for d := start; d.Before(today); d = d.AddDate(0, 0, 1) {
		// Skip days that are already cached
		if a.cache.Has(d.Format("2006-01-02")) {
			continue
		}

		// Get the APOD for the day
		_, err := a.Get(d.Format("2006-01-02"))
		if err != nil {
			log.Println("Error getting APOD:", err)
			continue
		}
		time.Sleep(1 * time.Second)
	}

	log.Println("Finished filling cache!!")
}

// Today gets today's APOD from the NASA API
func (a *APOD) Today() (resp *Response, err error) {
	resp, err = a.Get(a.TodaysDate())
	return resp, err
}

// TodaysDate return todays date in the format for apod.Get()
func (a *APOD) TodaysDate() string {
	return time.Now().UTC().Format("2006-01-02")
}

// RandomDate returns a random valid date for apod.Get()
func (a *APOD) RandomDate() string {
	// Must be after 1995-06-16 (first APOD) and before today
	start := time.Date(1995, 6, 16, 0, 0, 0, 0, time.UTC)
	end := time.Now().UTC()

	// Get a random date between start and end
	diff := end.Sub(start)
	random := time.Duration(rand.Int63n(int64(diff)))
	return start.Add(random).Format("2006-01-02")
}

// Random gets a random APOD from the NASA API
func (a *APOD) Random() (*Response, error) {
	return a.Get(a.RandomDate())
}

// IsValidDate checks if a date is valid for apod.Get()
func (a *APOD) IsValidDate(date string) bool {
	// Check if the date is in the correct format
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false
	}

	// Check if the date is after 1995-06-16 (first APOD)
	start := time.Date(1995, 6, 16, 0, 0, 0, 0, time.UTC)
	d, _ := time.Parse("2006-01-02", date)
	return d.After(start)
}

// GetImage gets an image from the image cache
func (a *APOD) GetImage(day string) (*ImageWrapper, bool) {
	return a.ImageCache.Get(day)
}

// Response is a single JSON response from the APOD API.
type Response struct {
	Title       string `json:"title"`
	Date        string `json:"date"`
	URL         string `json:"url"`
	HDURL       string `json:"hdurl"`
	MediaType   string `json:"media_type"`
	Explanation string `json:"explanation"`
	Thumbnail   string `json:"thumbnail_url"`
	Copyright   string `json:"copyright"`
	Service     string `json:"service_version"`
}

func (a *Response) String() string {
	return fmt.Sprintf("Title: %s\nDate: %s\nURL: %s\nHDURL: %s\nMediaType: %s\nExplanation: %s\nThumbnail: %s\nCopyright: %s\nService: %s", a.Title, a.Date, a.URL, a.HDURL, a.MediaType, a.Explanation, a.Thumbnail, a.Copyright, a.Service)
}

// ToEmbed converts an APODResponse to an embed for Discord
func (a *Response) ToEmbed(image []byte, format string) (*discordgo.MessageEmbed, *discordgo.File) {
	log.Println("Creating embed for", a.Date, "with format", format)

	embed := &discordgo.MessageEmbed{
		Title: a.Title,
		Color: 0xFF0000,
		Author: &discordgo.MessageEmbedAuthor{
			Name: a.Copyright,
		},
		// a.Date is in the format yyyy-mm-dd
		// but the url format is apyymmdd
		Description: fmt.Sprintf("[%s](https://apod.nasa.gov/apod/ap%s.html)\n", a.Date, strings.Replace(a.Date, "-", "", -1)[2:]),
	}

	filename := fmt.Sprintf("%s.%s", a.Date, format)
	embed.Image = &discordgo.MessageEmbedImage{
		URL: fmt.Sprintf("attachment://%s", filename),
	}

	if a.MediaType == "video" {
		if a.HDURL != "" {
			embed.Description += "VIDEO: " + a.HDURL
		} else {
			embed.Description += "VIDEO: " + a.URL
		}
	}

	return embed, &discordgo.File{
		Name:   filename,
		Reader: bytes.NewReader(image),
	}
}

// CreateExplanation creates a markdown formatted explanation of today's APOD
func (a *Response) CreateExplanation() string {
	return fmt.Sprintf("_%s_\n> %s", a.Title, a.Explanation)
}

// DownloadRawImage downloads the image without resizing
func (a *Response) DownloadRawImage() (*ImageWrapper, error) {
	log.Println("Downloading image for", a.Date)
	if a.MediaType == "image" {
		return downloadImage(a.HDURL)
	}
	return downloadImage(a.Thumbnail)
}

// DownloadSizedImage downloads the image and resizes it to be under the max size
func (a *Response) DownloadSizedImage() (*ImageWrapper, error) {
	img, err := a.DownloadRawImage()
	if err != nil {
		return img, err
	}

	err = img.Resize(DiscordMaxImageSize)
	return img, err
}
