package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var (
	zero = float64(0)
)

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "today",
		Description: "Get today's APOD.",
		Type:        discordgo.ChatApplicationCommand,
	},
	{
		Name:        "schedule",
		Description: "Schedule when to send APODs.\n",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{{
			Name:        "hour",
			Description: "The hour (utc) to send the APODs.\n",
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
}

var handlers = map[string]func(*discordgo.Session, *discordgo.InteractionCreate){
	"today": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		today, err := apod.Today()
		if err != nil {
			sendError(s, i, err)
			return
		}

		sendEmbeds(s, i, today.ToEmbed())
	},
	"schedule": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		allowed, err := authorize(s, i)

		if err != nil {
			sendError(s, i, err)
			return
		}

		if !allowed {
			sendMessage(s, i, "You do not have permission to use this command.")
			return
		}

		for _, option := range i.ApplicationCommandData().Options {
			if option.Name == "hour" {
				hour := int(option.Value.(float64))
				apod.Schedule(i.Interaction.ChannelID, hour)
				sendMessage(s, i, fmt.Sprintf("Astronomy picture of the day will be sent daily at %d:00 UTC. Use `/stop` to stop", hour))
				return
			}
		}
	},
	"stop": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		allowed, err := authorize(s, i)

		if err != nil {
			sendError(s, i, err)
			return
		}

		if !allowed {
			sendMessage(s, i, "You do not have permission to use this command.")
			return
		}

		apod.Stop(i.Interaction.ChannelID)
		sendMessage(s, i, "This channels scheduled astronomy picture of the day will no longer be sent.")
	},
}

// As of right now a user must have "Manage Server" permission (or higher) to use the bot.
const bitmask = discordgo.PermissionManageServer | discordgo.PermissionAll | discordgo.PermissionAdministrator

// authorize is a helper funciton to check if the user is authorized to use the bot.
func authorize(s *discordgo.Session, i *discordgo.InteractionCreate) (bool, error) {
	// check
	for _, id := range i.Interaction.Member.Roles {
		// get the role info
		role, err := s.State.Role(i.GuildID, id)
		if err != nil {
			return false, err
		}

		if role.Permissions&bitmask != 0 {
			return true, nil
		}
	}

	return false, nil
}

func sendMessage(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})

	if err != nil {
		fmt.Println("Error responding to interaction: ", err)
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
		fmt.Println("Error responding to interaction: ", err)
	}
}

func sendEmbeds(s *discordgo.Session, i *discordgo.InteractionCreate, embeds ...*discordgo.MessageEmbed) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: embeds,
		},
	})

	if err != nil {
		fmt.Println("Error responding to interaction: ", err)
	}
}
