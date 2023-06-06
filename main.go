package main

import (
	"log"
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
	log.Println("Starting APOD bot...")
	godotenv.Load()

	// Load tokens from .env file.
	discordToken := os.Getenv("DISCORD_TOKEN")
	apodToken := os.Getenv("APOD_TOKEN")

	if discordToken == "" || apodToken == "" {
		log.Println("Please set DISCORD_TOKEN and APOD_TOKEN in the .env file.")
		return
	}

	// Create a new Discord session using the provided bot token.
	session, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		log.Println("Error creating Discord session: ", err)
		return
	}

	// open the bolt key value store
	db, err := bolt.Open("./apod.db", 0600, &bolt.Options{Timeout: time.Second})

	if err != nil {
		log.Println("Error opening bolt db: ", err)
		return
	}
	defer db.Close()

	// create the schedule bucket
	db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("schedule"))
		return nil
	})

	apod = &APOD{
		key:     apodToken,
		db:      db,
		session: session,
	}

	// cache the current APOD response
	_, err = apod.Today()
	if err != nil {
		log.Println("Error caching APOD: ", err)
	}

	// number of scheduled APODs
	scheduled := 0
	apod.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("schedule"))
		c := b.Cursor()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			scheduled++
		}

		return nil
	})
	log.Println("Schedule size: ", scheduled)

	var guilds []string
	ch := make(chan struct{})
	session.AddHandler(func(s *discordgo.Session, event *discordgo.Ready) {
		log.Println("Guilds: ", len(event.Guilds))

		for _, guild := range event.Guilds {
			guilds = append(guilds, guild.ID)
		}

		log.Println("Bot is ready.")
		ch <- struct{}{}
	})

	session.Open()

	<-ch

	// Handle application commands
	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if handler, ok := handlers[i.ApplicationCommandData().Name]; ok {
			log.Println("Handling command: ", i.ApplicationCommandData().Name)
			handler(s, i)
		}
	})

	// Announce when the bot joins a guild.
	session.AddHandler(func(s *discordgo.Session, event *discordgo.GuildCreate) {
		// Check if the bot was already in the guild
		for _, guild := range guilds {
			if guild == event.ID {
				return
			}
		}

		log.Printf("Joined server: %s %q\n", event.ID, event.Name)
	})

	// Announce when the bot is removed from a guild.
	session.AddHandler(func(s *discordgo.Session, event *discordgo.GuildDelete) {
		if !event.Guild.Unavailable {
			log.Println("Left server: ", event.ID)

			// update the bot to check it still has access to all channels
			apod.UpdateSchedule()
		}
	})

	// Update the bot's interactions
	_, err = session.ApplicationCommandBulkOverwrite(session.State.User.ID, "", commands)
	if err != nil {
		log.Println("Error overwriting commands: ", err)
	}

	log.Println("Bot is running. Press CTRL-C to exit.")
	apod.RunScheduler()
}
