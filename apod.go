package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	bolt "go.etcd.io/bbolt"
)

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

	Image []byte
}

func (a *APODResponse) String() string {
	return fmt.Sprintf("Title: %s\nDate: %s\nURL: %s\nHDURL: %s\nMediaType: %s\nExplanation: %s\nThumbnail: %s\nCopyright: %s\nService: %s", a.Title, a.Date, a.URL, a.HDURL, a.MediaType, a.Explanation, a.Thumbnail, a.Copyright, a.Service)
}

// Convert a APODResponse to an embed
func (a *APODResponse) ToEmbed() (*discordgo.MessageEmbed, *discordgo.File) {
	embed := &discordgo.MessageEmbed{
		Title: a.Title,
		Color: 0xFF0000,
		Author: &discordgo.MessageEmbedAuthor{
			Name: a.Copyright,
		},
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
			embed.Description = "VIDEO: " + a.HDURL
		} else {
			embed.Description = "VIDEO: " + a.URL
		}
	}

	return embed, &discordgo.File{
		Name:   filename,
		Reader: bytes.NewReader(a.Image),
	}
}

func (a *APODResponse) CreateExplaination() string {
	if a.MediaType == "image" {
		return fmt.Sprintf("_%s_\n> %s", a.Title, a.Explanation)
	} else if a.MediaType == "video"{
		return fmt.Sprintf("_%s_\n%s\n> %s", a.Title, a.URL, a.Explanation)
	}

	return ""
}

type APOD struct {
	key     string
	db      *bolt.DB
	session *discordgo.Session

	// cache holds the APODResponse for the current day.
	cache APODResponse

	// schedule maps channel IDs to the hour of the day to send a message
	schedule map[string]int
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

	// If the response is an image, download it
	if response.MediaType == "image" {
		response.Image, err = downloadImage(response.HDURL)
		if err != nil {
			return response, err
		}

		fmt.Println("Downloaded image", len(response.Image))
	} else {
		response.Image, err = downloadImage(response.Thumbnail)
		if err != nil {
			return response, err
		}

		fmt.Println("Downloaded thumbnail", len(response.Image))
	}

	// Cache the response
	a.cache = response

	return response, nil
}

func downloadImage(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

// Schedule adds a job to the scheduler to send an APOD message to a channel
// at a specific hour of the day (in UTC)
func (apod *APOD) Schedule(channel string, hour int) {
	apod.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("schedule"))
		h := fmt.Sprintf("%d", hour)
		b.Put([]byte(channel), []byte(h))
		return nil
	})
}

func (apod *APOD) Stop(channel string) {
	apod.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("schedule"))
		b.Delete([]byte(channel))
		return nil
	})
}

func (apod *APOD) RunScheduler() {
	fmt.Println("Starting APOD scheduler")

	// Every hour on the hour check if we need to send an APOD message
	for {
		sleepUntilNextHour()

		// Map from channelID to hour of the day to send a message (in UTC)
		apod.schedule = make(map[string]int)

		// Get all the channels that have a scheduled APOD
		apod.db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("schedule"))
			c := b.Cursor()

			for k, v := c.First(); k != nil; k, v = c.Next() {
				// Get the hour
				hour, err := strconv.Atoi(string(v))
				if err != nil {
					return err
				}

				// Get the channel
				channel := string(k)

				apod.schedule[channel] = hour
			}

			return nil
		})

		// Get the current time
		now := time.Now().UTC()

		// Get the hour of the day
		hour := now.Hour()

		fmt.Printf("Checking for APOD messages for hour %d...\n", hour)

		// Prepare the message
		res, err := apod.Today()
		if err != nil {
			fmt.Println(err)
			continue
		}

		embed, image := res.ToEmbed()

		// Check each channel
		for channelID, hourToSend := range apod.schedule {
			// If the hour matches, send the message
			if hour == hourToSend {
				fmt.Println("Sending APOD message to " + channelID)

				// Send the message to the channel
				_, err = apod.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
					Embeds: []*discordgo.MessageEmbed{embed},
					Files:  []*discordgo.File{image},
				})
				if err != nil {
					fmt.Println("Error sending message:", err)
					delete(apod.schedule, channelID) // Remove the channel from the schedule
				}
			}
		}
	}
}

func sleepUntilNextHour() {
	now := time.Now().UTC()
	next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, time.UTC)
	time.Sleep(next.Sub(now))
}
