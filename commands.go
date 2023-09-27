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

// Abstracts APOD Response commands "today", "random", and "date" using an
// arbitrary function that returns an APODResponse and an error.
func (bot *Bot) get(s *discordgo.Session, i *discordgo.InteractionCreate, date string) {
	// Check if the date is valid and if not, send an error message.
	if !bot.apod.IsValidDate(date) {
		sendError(s, i, fmt.Errorf("\"%s\" is not a valid date. Use yyyy-mm-dd format, and choose a date after 1995-06-16", date))
		return
	}

	// Let the user know we are working on it.
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	if err != nil {
		log.Println("failed to prepare response: ", err)
		finalizeMessage(s, i, "Sorry, something went wrong!")
		return
	}

	today, err := bot.apod.Get(date)
	if err != nil {
		finalizeError(s, i, err)
		return
	}

	image, format, err := GetOrSet(bot.apod.imageCache, today.Date, today.DownloadSizedImage)
	if err != nil {
		finalizeError(s, i, err)
		return
	}

	embed, file := today.ToEmbed(image, format)
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
		Files:  []*discordgo.File{file},
	})

	if err != nil {
		log.Println("sendEmbed error responding to interaction: ", err)
	}

	bot.db.Sent(i.ChannelID, today.Date)
}

// handler handles application commands, switching on the command name
func (bot *Bot) handler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Println("Command: ", i.ApplicationCommandData().Name)

	switch i.ApplicationCommandData().Name {
	case "today":
		date := bot.apod.TodaysDate()
		bot.get(s, i, date)
	case "random":
		date := bot.apod.RandomDate()
		bot.get(s, i, date)
	case "specific":
		var date string
		for _, option := range i.ApplicationCommandData().Options {
			if option.Name == "date" {
				date = option.Value.(string)
			}
		}

		bot.get(s, i, date)
	case "explanation":
		// Get the last APOD sent to this channel
		var apod *APODResponse
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
			log.Println("failed to send unknown command warning: ", err)
		}
	}
}

// As of right now a user must have "Manage Server" permission (or higher) to use the bot.
const bitmask = discordgo.PermissionManageServer | discordgo.PermissionAdministrator

// sendMessage responds to a new interaction that hasn't been deferred
func sendMessage(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})

	if err != nil {
		log.Println("sendMessage error responding to interaction: ", err)
	}
}

// finalizeMessage responds to a deferred interaction created with
// "InteractionResponseDeferredChannelMessageWithSource"
func finalizeMessage(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})

	if err != nil {
		log.Println("finalizeMessage error responding to interaction: ", err)
	}
}

func sendError(s *discordgo.Session, i *discordgo.InteractionCreate, e error) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: e.Error(),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	if err != nil {
		log.Println("sendError error responding to interaction: ", err)
	}
}

func finalizeError(s *discordgo.Session, i *discordgo.InteractionCreate, e error) {
	finalizeMessage(s, i, e.Error())
}
