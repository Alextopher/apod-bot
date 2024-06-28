package main

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/Alextopher/apod-bot/internal/apod"
	"github.com/bwmarrin/discordgo"
)

// As of right now a user must have "Manage Server" permission (or higher) to use the bot.
const bitmask = discordgo.PermissionManageServer | discordgo.PermissionAdministrator

var (
	zero = float64(0)
)

const (
	none      = 0
	ephemeral = discordgo.MessageFlagsEphemeral
)

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "today",
		Description: "Get today's APOD",
		Type:        discordgo.ChatApplicationCommand,
	},
	{
		Name:        "random",
		Description: "Get a random APOD",
		Type:        discordgo.ChatApplicationCommand,
	},
	{
		Name:        "specific",
		Description: "Get a specific APOD",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{{
			Name:        "date",
			Description: "In yyyy-mm-dd format",
			Type:        discordgo.ApplicationCommandOptionString,
			Required:    true,
		}},
	},
	{
		Name:        "explanation",
		Description: "Get the explanation of the last APOD",
		Type:        discordgo.ChatApplicationCommand,
	},
	{
		Name:        "schedule",
		Description: "Schedule when to send APODs\n",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{{
			Name:        "hour",
			Description: "The hour (utc) to send the APODs\n",
			Type:        discordgo.ApplicationCommandOptionInteger,
			MinValue:    &zero,
			MaxValue:    23,
			Required:    true,
		}},
	},
	{
		Name:        "stop",
		Description: "Stop sending APODs.\n",
		Type:        discordgo.ChatApplicationCommand,
	},
	{
		Name:        "source",
		Description: "Visit the bot's github repo",
		Type:        discordgo.ChatApplicationCommand,
	},
}

// Responds to an interaction with an APOD
func (bot *Bot) get(msg *Response, resp *apod.Response) {
	embed, file := bot.ToEmbed(resp)
	if embed == nil || file == nil {
		msg.TextMessage("Error creating embed", ephemeral)
		return
	}

	bot.db.Sent(msg.interaction.ChannelID, resp.Date)
	err := msg.EmbedMessage(embed, file, none)
	if err != nil {
		log.Println("Error sending message:", err)
	}
}

// commandHandler handles application commands, switching on the command name
func (bot *Bot) commandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Println("Command: ", i.ApplicationCommandData().Name)

	switch i.ApplicationCommandData().Name {
	case "today":
		msg := NewResponse(s, i.Interaction, none)
		resp, err := apod.Retry(func() (*apod.Response, error) {
			return bot.apod.Today()
		})
		if err != nil {
			msg.TextMessage("Error getting random APOD: "+err.Error(), ephemeral)
			return
		}
		bot.get(msg, resp)
	case "random":
		msg := NewResponse(s, i.Interaction, none)
		resp, err := apod.Retry(func() (*apod.Response, error) {
			return bot.apod.Random()
		})
		if err != nil {
			msg.TextMessage("Error getting random APOD: "+err.Error(), ephemeral)
			return
		}
		bot.get(msg, resp)
	case "specific":
		msg := NewResponse(s, i.Interaction, none)

		var date string
		for _, option := range i.ApplicationCommandData().Options {
			if option.Name == "date" {
				date = option.Value.(string)
			}
		}

		resp, err := apod.Retry(func() (*apod.Response, error) {
			return bot.apod.Get(date)
		})
		if err != nil {
			msg.TextMessage("Error getting random APOD: "+err.Error(), ephemeral)
			return
		}
		bot.get(msg, resp)
	case "explanation":
		// Get the last APOD sent to this channel
		var apod *apod.Response
		var err error

		msg := NewResponse(s, i.Interaction, none)
		if date, ok := bot.db.GetLast(i.ChannelID); ok {
			apod, err = bot.apod.Get(date)
			if err != nil {
				msg.TextMessage("Sorry, I couldn't get the explanation for the last APOD", ephemeral)
				log.Println("Error getting explanation: ", err)
				return
			}
		} else {
			apod, err = bot.apod.Today()
			if err != nil {
				msg.TextMessage("Sorry, I couldn't get the explanation for today's APOD", ephemeral)
				log.Println("Error getting explanation: ", err)
				return
			}
		}

		msg.TextMessage(apod.CreateExplanation(), none)
	case "schedule":
		msg := NewResponse(s, i.Interaction, ephemeral)

		allowed := i.Interaction.Member.Permissions&bitmask != 0
		if !allowed {
			msg.TextMessage("You must have \"Manage Server\" permissions or higher.", ephemeral)
			return
		}

		for _, option := range i.ApplicationCommandData().Options {
			if option.Name == "hour" {
				hour := int(option.Value.(float64))
				bot.db.Set(i.ChannelID, hour)
				msg.TextMessage(fmt.Sprintf("Astronomy picture of the day will be sent daily at %d:00 UTC. Use `/stop` to stop", hour), none)
				return
			}
		}
	case "stop":
		msg := NewResponse(s, i.Interaction, ephemeral)

		allowed := i.Interaction.Member.Permissions&bitmask != 0
		if !allowed {
			msg.TextMessage("You must have \"Manage Server\" permissions or higher.", ephemeral)
			return
		}

		bot.db.Remove(i.ChannelID)
		msg.TextMessage("This channels scheduled astronomy picture of the day will no longer be sent.", none)
	case "source":
		msg := NewResponse(s, i.Interaction, none)
		msg.TextMessage("https://github.com/Alextopher/apod-bot", none)
	default:
		log.Println("Unknown command: ", i.ApplicationCommandData().Name)
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Unknown command: " + i.ApplicationCommandData().Name,
			},
		})
		if err != nil {
			log.Println("failed to send unknown command warning: ", err)
		}
	}
}

// ToEmbed creates a discordgo.MessageEmbed from an APOD response
func (bot *Bot) ToEmbed(a *apod.Response) (*discordgo.MessageEmbed, *discordgo.File) {
	// Get the image and resize it for discord
	image, err := bot.apod.GetImage(a.Date)
	if err != nil {
		log.Println("Error getting image for", a.Date, ":", err)
		return nil, nil
	}

	err = image.Resize(discordMaxImageSize)
	if err != nil {
		log.Println("Error resizing image for", a.Date, ":", err)
		return nil, nil
	}

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

	filename := fmt.Sprintf("%s.%s", a.Date, image.Format)
	embed.Image = &discordgo.MessageEmbedImage{
		URL: fmt.Sprintf("attachment://%s", filename),
	}

	if a.MediaType == "video" {
		if a.HdURL != "" {
			embed.Description += "VIDEO: " + a.HdURL
		} else {
			embed.Description += "VIDEO: " + a.URL
		}
	}

	return embed, &discordgo.File{
		Name:   filename,
		Reader: bytes.NewReader(image.Bytes),
	}
}
