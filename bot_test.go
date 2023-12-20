package main

// Test sending dm to owner
import (
	"log"
	"os"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func TestMessageOwner(t *testing.T) {
	key, ok := os.LookupEnv("DISCORD_TOKEN")
	if !ok {
		godotenv.Load()
		key = os.Getenv("DISCORD_TOKEN")
	}

	owner, ok := os.LookupEnv("OWNER")
	if !ok {
		godotenv.Load()
		owner = os.Getenv("OWNER")
	}

	if key == "" || owner == "" {
		t.Error("Please set DISCORD_TOKEN and OWNER in the .env file.")
		return
	}

	// Create a new Discord session using the provided bot token.
	session, err := discordgo.New("Bot " + key)
	if err != nil {
		log.Println("Error creating Discord session: ", err)
		return
	}

	bot := &Bot{
		session: session,
	}

	ch := make(chan struct{})
	session.AddHandler(func(s *discordgo.Session, event *discordgo.Ready) {
		log.Println("Bot is ready.")
		ch <- struct{}{}
	})
	session.Open()
	<-ch

	err = bot.SetOwner(owner)
	if err != nil {
		t.Error(err)
	}

	err = bot.MessageOwner("Test message")
	if err != nil {
		t.Error(err)
	}
}
