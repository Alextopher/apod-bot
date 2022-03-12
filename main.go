package main

import (
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var apod *APOD

func main() {
	godotenv.Load()

	// Load tokens from .env file.
	discordToken := os.Getenv("DISCORD_TOKEN")
	apodToken := os.Getenv("APOD_TOKEN")

	if discordToken == "" || apodToken == "" {
		fmt.Println("Please set DISCORD_TOKEN and APOD_TOKEN in .env file.")
		return
	}

	apod = &APOD{
		key: apodToken,
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	// Wait until the bot is ready.
	ch := make(chan struct{})
	dg.AddHandler(func(s *discordgo.Session, event *discordgo.Ready) {
		fmt.Println("Bot is ready.")
		ch <- struct{}{}
	})

	dg.Open()
	<-ch

	_, err = dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, "", commands)
	if err != nil {
		fmt.Println("Error overwriting commands: ", err)
	}

	// Handle application commands
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if handler, ok := handlers[i.ApplicationCommandData().Name]; ok {
			handler(s, i)
		}
	})

	select {}
}
