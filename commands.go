package main

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

var (
	zero = float64(0)
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

func (bot *Bot) handler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Println("Command: ", i.ApplicationCommandData().Name)

	switch i.ApplicationCommandData().Name {
	case "today":
		// Let the user know we are working on it.
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			log.Println("Error responding to interaction: ", err)
		}

		today, err := bot.apod.Today()
		if err != nil {
			sendError(s, i, err)
			return
		}

		image, err := bot.apod.imageCache.GetOrSet(today.Date, today.DownloadImage)
		if err != nil {
			sendError(s, i, err)
			return
		}

		embed, file := today.ToEmbed(image)
		sendEmbed(s, i.Interaction, []*discordgo.MessageEmbed{embed}, []*discordgo.File{file})
		bot.db.Sent(i.ChannelID, today.Date)
	case "random":
		// Let the user know we are working on it.
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			log.Println("Error responding to interaction: ", err)
		}

		random, err := bot.apod.Random()
		if err != nil {
			sendError(s, i, err)
			return
		}

		image, err := random.DownloadImage()
		if err != nil {
			sendError(s, i, err)
			return
		}

		embed, file := random.ToEmbed(image)
		sendEmbed(s, i.Interaction, []*discordgo.MessageEmbed{embed}, []*discordgo.File{file})
		bot.db.Sent(i.ChannelID, random.Date)
	case "explanation":
		// Get the last APOD sent to this channel
		var apod APODResponse
		var err error

		if date, ok := bot.db.GetLast(i.ChannelID); ok {
			apod, err = bot.apod.Get(date)
			if err != nil {
				sendError(s, i, err)
				return
			}
		} else {
			apod, err = bot.apod.Today()
			if err != nil {
				sendError(s, i, err)
				return
			}
		}

		sendMessage(s, i, apod.CreateExplanation())
	case "schedule":
		allowed := i.Interaction.Member.Permissions&bitmask != 0
		if !allowed {
			sendMessage(s, i, "You must have \"Manage Server\" permissions or higher.")
			return
		}

		for _, option := range i.ApplicationCommandData().Options {
			if option.Name == "hour" {
				hour := int(option.Value.(float64))
				bot.db.Set(i.ChannelID, hour)
				sendMessage(s, i, fmt.Sprintf("Astronomy picture of the day will be sent daily at %d:00 UTC. Use `/stop` to stop", hour))
				return
			}
		}
	case "stop":
		allowed := i.Interaction.Member.Permissions&bitmask != 0
		if !allowed {
			sendMessage(s, i, "You must have \"Manage Server\" permissions or higher.")
			return
		}

		bot.db.Remove(i.ChannelID)
		sendMessage(s, i, "This channels scheduled astronomy picture of the day will no longer be sent.")
	case "source":
		sendMessage(s, i, "https://github.com/Alextopher/apod-bot")
	default:
		log.Println("Unknown command: ", i.ApplicationCommandData().Name)
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Unknown command: " + i.ApplicationCommandData().Name,
			},
		})
		if err != nil {
			log.Println("Error responding to interaction: ", err)
		}
	}
}

// As of right now a user must have "Manage Server" permission (or higher) to use the bot.
const bitmask = discordgo.PermissionManageServer | discordgo.PermissionAdministrator

func sendMessage(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})

	if err != nil {
		log.Println("Error responding to interaction: ", err)
	}
}

func sendError(s *discordgo.Session, i *discordgo.InteractionCreate, e error) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: e.Error(),
		},
	})

	if err != nil {
		log.Println("Error responding to interaction: ", err)
	}
}

func sendEmbed(s *discordgo.Session, i *discordgo.Interaction, embeds []*discordgo.MessageEmbed, files []*discordgo.File) {
	_, err := s.InteractionResponseEdit(i, &discordgo.WebhookEdit{
		Embeds: &embeds,
		Files:  files,
	})

	if err != nil {
		log.Println("Error responding to interaction: ", err)
	}
}
