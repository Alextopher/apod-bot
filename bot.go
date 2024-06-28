package main

import (
	"log"
	"time"

	"github.com/Alextopher/apod-bot/internal/apod"
	"github.com/bwmarrin/discordgo"
)

const discordMaxImageSize = 8 * 1024 * 1024

// Bot is the discord bot
type Bot struct {
	db   *DB
	apod *apod.APOD

	session *discordgo.Session
	owner   *discordgo.User
}

// SetOwner sets the bot's owner
func (b *Bot) SetOwner(ownerID string) error {
	owner, err := b.session.User(ownerID)
	if err != nil {
		return err
	}
	b.owner = owner
	return nil
}

// MessageOwner sends a message to the bot's owner
func (b *Bot) MessageOwner(msg string) error {
	if b.owner == nil {
		return nil
	}

	channel, err := b.session.UserChannelCreate(b.owner.ID)
	if err != nil {
		return err
	}

	_, err = b.session.ChannelMessageSend(channel.ID, msg)
	if err != nil {
		return err
	}

	return nil
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
		time.Sleep(time.Minute)

		// Prepare the message with retries
		res, err := apod.Retry(func() (*apod.Response, error) {
			return b.apod.Today()
		})

		if err != nil {
			log.Println("scheduler: error getting today's APOD:", err)
			continue
		}

		embed, file := b.ToEmbed(res)

		hour := time.Now().UTC().Hour()
		b.db.View(func(channelID string, hourToSend int) {
			if hour == hourToSend {
				log.Printf("scheduler: sending APOD to %s\n", channelID)

				_, err = b.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
					Embeds: []*discordgo.MessageEmbed{embed},
					Files:  []*discordgo.File{file},
				})

				if err != nil {
					log.Println("scheduler: error sending message:", err)
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
