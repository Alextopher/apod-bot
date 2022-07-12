package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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

	// Cache the response
	a.cache = response

	return response, nil
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

	// Every hour on the hour check if we need to send an APOD message
	for {
		sleepUntilNextHour()

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

		embed := res.ToEmbed()

		// Check each channel
		for channelID, hourToSend := range apod.schedule {
			fmt.Println("Checking channel", channelID, "hour", hourToSend)

			// If the hour matches, send the message
			if hour == hourToSend {
				fmt.Println("Sending APOD message to " + channelID)

				// Send the message to the channel
				_, err = apod.session.ChannelMessageSendEmbed(channelID, embed)
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
