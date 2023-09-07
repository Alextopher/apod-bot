package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	_ "image/jpeg"

	"github.com/bwmarrin/discordgo"
)

// APOD is the bot's APOD API client
//
// It keeps a cache of the current day's APOD
type APOD struct {
	key        string
	cache      *APODCache
	imageCache *ImageCache
}

// Get the APOD response for a specific date
func (a *APOD) Get(date string) (response APODResponse, err error) {
	// If the cache has the response, return that
	if resp, ok := a.cache.Get(date); ok {
		return resp, err
	}

	// Get the JSON response from the API
	req := fmt.Sprintf("https://api.nasa.gov/planetary/apod?thumbs=true&date=%s&api_key=%s", date, a.key)
	resp, err := http.Get(req)

	// Check for error codes
	if resp.StatusCode != http.StatusOK {
		return response, fmt.Errorf("NASA API Failure: %s", resp.Status)
	}
	if err != nil {
		return response, err
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

// Today gets today's APOD from the NASA API
func (a *APOD) Today() (APODResponse, error) {
	date := time.Now().Format("2006-01-02")
	return a.Get(date)
}

// Gets a random APOD from the NASA API
func (a *APOD) Random() (APODResponse, error) {
	// Must be after 1995-06-16 (first APOD) and before today
	start, err := time.Parse("2006-01-02", "1995-06-16")
	if err != nil {
		return APODResponse{}, err
	}
	end := time.Now()

	// Get a random date between start and end
	diff := end.Sub(start)
	random := time.Duration(rand.Int63n(int64(diff)))
	date := start.Add(random).Format("2006-01-02")

	return a.Get(date)
}

// APODResponse is a single JSON response from the APOD API.
type APODResponse struct {
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

func (a *APODResponse) String() string {
	return fmt.Sprintf("Title: %s\nDate: %s\nURL: %s\nHDURL: %s\nMediaType: %s\nExplanation: %s\nThumbnail: %s\nCopyright: %s\nService: %s", a.Title, a.Date, a.URL, a.HDURL, a.MediaType, a.Explanation, a.Thumbnail, a.Copyright, a.Service)
}

// ToEmbed converts an APODResponse to an embed for Discord
func (a *APODResponse) ToEmbed(image []byte) (*discordgo.MessageEmbed, *discordgo.File) {
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

	var url string
	if a.MediaType == "image" {
		if a.HDURL != "" {
			url = a.HDURL
		} else {
			url = a.URL
		}
	} else if a.MediaType == "video" {
		url = a.Thumbnail
	}

	// The filename of the image is the last part of the URL
	parts := strings.Split(url, "/")
	filename := parts[len(parts)-1]

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
func (a *APODResponse) CreateExplanation() string {
	return fmt.Sprintf("_%s_\n> %s", a.Title, a.Explanation)
}

// Adds the image to the response
func (a *APODResponse) DownloadImage() (image []byte, err error) {
	log.Println("Downloading image for", a.Date)
	if a.MediaType == "image" {
		return downloadImage(a.HDURL)

	} else {
		return downloadImage(a.Thumbnail)
	}
}

// Checks if the response is for today
func (a *APODResponse) IsToday() bool {
	return a.Date == time.Now().Format("2006-01-02")
}
