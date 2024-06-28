package main

import (
	"log"
	"os"

	"github.com/Alextopher/apod-bot/internal/apod"
	"github.com/Alextopher/apod-bot/internal/cache"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	// Load tokens from .env file.
	apodToken := os.Getenv("APOD_TOKEN")

	// Connect to APOD API
	cacheFile, err := os.OpenFile("apod.cache", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Println("Error opening apod.cache: ", err)
		return
	}

	apodCache, err := cache.NewAppendCache[*apod.Response](cacheFile, cacheFile)
	if err != nil {
		log.Println("Error creating cache: ", err)
		return
	}

	imageCache := apod.NewImageCache("images")
	a := apod.NewClient(apodToken, apodCache, imageCache)

	// Get all apods from 1995-06-16 to today
	a.Fill()
}
