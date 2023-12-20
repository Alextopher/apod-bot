package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	"github.com/Alextopher/apod-bot/internal/apod"
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

	// Create reader and writer for the database
	f, err := os.OpenFile("apod.db", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Println("Error opening apod.db: ", err)
		return
	}
	defer f.Close()

	db, err := NewDB(f, f)
	if err != nil {
		log.Println("Error creating database: ", err)
		return
	}

	// Connect to APOD API
	cacheFile, err := os.OpenFile("apod.cache", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Println("Error opening apod.cache: ", err)
		return
	}

	cache, err := apod.NewAPODCache(cacheFile, cacheFile)
	if err != nil {
		log.Println("Error creating cache: ", err)
		return
	}

	imageCache, err := apod.NewDirectoryImageCache("images")
	if err != nil {
		log.Println("Error creating image cache: ", err)
		return
	}

	bot := &Bot{
		db:      db,
		apod:    apod.NewClient(apodToken, cache, imageCache),
		session: session,
	}

	// Set the bot's owner
	if owner, ok := os.LookupEnv("OWNER"); ok {
		err = bot.SetOwner(owner)
		if err != nil {
			log.Println("Error setting bot owner: ", err)
			return
		}
	}

	// cache the current APOD response
	_, err = bot.apod.Today()
	if err != nil {
		log.Println("Error caching APOD: ", err)
	}

	// number of scheduled APODs
	log.Println("Schedule size: ", bot.db.Size())

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
	session.AddHandler(bot.handler)

	// Announce when the bot joins a guild.
	session.AddHandler(func(s *discordgo.Session, event *discordgo.GuildCreate) {
		// Check if the bot was already in the guild
		for _, guild := range guilds {
			if guild == event.ID {
				return
			}
		}

		log.Printf("Joined server: %s %q\n", event.ID, event.Name)
		err = bot.MessageOwner(fmt.Sprintf("I was just added to %s %q", event.ID, event.Name))
		if err != nil {
			log.Println("Error messaging owner: ", err)
		}
	})

	// Announce when the bot is removed from a guild.
	session.AddHandler(func(s *discordgo.Session, event *discordgo.GuildDelete) {
		log.Println("Left server: ", event.ID)

		// update the bot to check it still has access to all channels
		bot.UpdateSchedule()
		err = bot.MessageOwner(fmt.Sprintf("I was just removed from %s", event.ID))
		if err != nil {
			log.Println("Error messaging owner: ", err)
		}

		// remove the id from the list of guilds
		for i, guild := range guilds {
			if guild == event.ID {
				guilds = append(guilds[:i], guilds[i+1:]...)
				return
			}
		}
	})

	// Update the bot's interactions
	_, err = session.ApplicationCommandBulkOverwrite(session.State.User.ID, "", commands)
	if err != nil {
		log.Println("Error overwriting commands: ", err)
	}

	log.Println("Bot is running. Press CTRL-C to exit.")
	go bot.RunScheduler()
	go bot.apod.Fill()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
}
