package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
)

type APOD struct {
	key string

	// cache holds the APODResponse for the current day.
	cache APODResponse
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

// APOD uses the NASA API to get todays astronomy picture of the day
func (a *APOD) Today() (APODResponse, error) {
	const baseURL = "https://api.nasa.gov/planetary/apod?thumbs=true&concept_tags=true&hd=true&api_key="

	var response APODResponse
	var err error

	// Get today's date
	date := time.Now().Format("2006-01-02")

	// If the cache has today's response, return it
	if a.cache.Date == date {
		return a.cache, nil
	}

	// Get the JSON response from the API
	resp, err := http.Get(baseURL + a.key)
	if err != nil {
		return response, err
	}

	// Decode the JSON response
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return response, err
	}

	// Cache the response
	a.cache = response

	return response, nil
}

// Convert a APODResponse to an embed
func (a APODResponse) ToEmbed() *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: a.Title,
		Color: 0xFF0000,
		Author: &discordgo.MessageEmbedAuthor{
			Name: a.Copyright,
		},
	}

	if a.MediaType == "image" {
		embed.Image = &discordgo.MessageEmbedImage{
			URL: a.HDURL,
		}
	} else {
		embed.Video = &discordgo.MessageEmbedVideo{
			URL: a.HDURL,
		}
	}

	return embed
}
