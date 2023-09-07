package main

import (
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	db   *DB
	apod *APOD

	session *discordgo.Session
	owner   *discordgo.User
}

// SetOwner sets the bot's owner
func (b *Bot) SetOwner(ownerID string) {
	owner, err := b.session.User(ownerID)
	if err != nil {
		log.Println("Error setting owner:", err)
		return
	}

	b.owner = owner
}

// Messages the bot's owner
func (b *Bot) MessageOwner(msg string) {
	if b.owner == nil {
		return
	}

	_, err := b.session.UserChannelCreate(b.owner.ID)
	if err != nil {
		log.Println("Error creating DM channel:", err)
		return
	}

	_, err = b.session.ChannelMessageSend(b.owner.ID, msg)
	if err != nil {
		log.Println("Error sending message to owner:", err)
	}
}

// Schedule adds a job to the scheduler to send an APOD message to a channel
// at a specific hour of the day (in UTC)
func (b *Bot) Schedule(channel string, hour int) {
	b.db.Set(channel, hour)
}

// Stop removes a server from the scheduler
func (b *Bot) Stop(channel string) {
	b.db.Remove(channel)
}

// UpdateSchedule checks if the bot has access to the channels in the schedule
// and removes any channels it doesn't have access to
func (b *Bot) UpdateSchedule() {
	b.db.RemoveIf(func(channelID string, _ int) bool {
		// Check if the bot has access to the channel
		_, err := b.session.Channel(channelID)
		if err != nil {
			log.Printf("Removed channel %s from the schedule\n", channelID)
			return true
		}

		return false
	})
}

// RunScheduler runs the scheduler, checking every hour on the hour if it needs
// to send an APOD message
func (b *Bot) RunScheduler() {
	b.UpdateSchedule()
	for {
		sleepUntilNextHour()

		// Prepare the message with retries
		var res APODResponse
		var err error
		backOff := time.Second

		for {
			res, err = b.apod.Today()
			if err == nil || backOff > time.Hour {
				break
			}

			log.Printf("Message prepare failed %s trying again in %s\n", err, backOff)
			time.Sleep(backOff)
			backOff *= 2
		}

		if err != nil {
			log.Println("Message prepare failed after multiple retries", err)
			return
		}

		image, err := b.apod.imageCache.GetOrSet(res.Date, res.DownloadImage)
		embed, file := res.ToEmbed(image)
		hour := time.Now().UTC().Hour()
		b.db.View(func(channelID string, hourToSend int) {
			if hour == hourToSend {
				log.Println("Sending APOD message to " + channelID)

				_, err = b.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
					Embeds: []*discordgo.MessageEmbed{embed},
					Files:  []*discordgo.File{file},
				})

				if err != nil {
					log.Println("Error sending message:", err)
				} else {
					b.db.Sent(channelID, res.Date)
				}
			}
		})
	}
}

func sleepUntilNextHour() {
	now := time.Now().UTC()
	next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, time.UTC)
	time.Sleep(next.Sub(now))
}