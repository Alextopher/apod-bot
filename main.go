package main

import (
	"fmt"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	bolt "go.etcd.io/bbolt"
)

var (
	apod *APOD
)

func main() {
	fmt.Println("Starting APOD bot...")
	godotenv.Load()

	// Load tokens from .env file.
	discordToken := os.Getenv("DISCORD_TOKEN")
	apodToken := os.Getenv("APOD_TOKEN")

	if discordToken == "" || apodToken == "" {
		fmt.Println("Please set DISCORD_TOKEN and APOD_TOKEN in .env file.")
		return
	}

	// Create a new Discord session using the provided bot token.
	session, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	ch := make(chan struct{})
	session.AddHandler(func(s *discordgo.Session, event *discordgo.Ready) {
		fmt.Println("Bot is ready.")
		ch <- struct{}{}
	})

	session.Open()

	// open the bolt key value store
	db, err := bolt.Open("./apod.db", 0600, &bolt.Options{Timeout: time.Second})

	if err != nil {
		fmt.Println("Error opening bolt db: ", err)
		return
	}
	defer db.Close()

	// create the schedule bucket
	db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("schedule"))
		return nil
	})

	// Handle application commands
	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if handler, ok := handlers[i.ApplicationCommandData().Name]; ok {
			handler(s, i)
		}
	})

	// Wait for the bot to be ready.
	<-ch

	// Update the bot's interactions
	_, err = session.ApplicationCommandBulkOverwrite(session.State.User.ID, "", commands)
	if err != nil {
		fmt.Println("Error overwriting commands: ", err)
	}

	apod = &APOD{
		key:     apodToken,
		db:      db,
		session: session,
	}

	apod.RunScheduler()
}
