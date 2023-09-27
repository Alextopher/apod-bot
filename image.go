package main

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"log"
	"net/http"

	_ "image/gif"
	_ "image/png"
)

// DiscordMaxImageSize is the maximum size (in bytes) of an image that can be sent to Discord
const DiscordMaxImageSize = 8 * 1024 * 1024

// resizeImage reads an image and resizes it to be close to max_size
func resizeImage(img []byte, maxSize int) ([]byte, string, error) {
	// Decode the image
	message, format, err := image.Decode(bytes.NewReader(img))
	if err != nil {
		log.Println("failed to decode image")
		return nil, format, err
	}

	buf := new(bytes.Buffer)

	// If the image is already small enough, return it
	if len(img) < maxSize {
		return img, format, nil
	}

	// Decrease the quality of the image until it fits within the max size
	for quality := 100; quality > 0; quality -= 5 {
		buf.Reset()
		err := jpeg.Encode(buf, message, &jpeg.Options{
			Quality: quality,
		})

		if err != nil {
			log.Println("could not encode image")
			return nil, "jpeg", err
		}

		if buf.Len() < maxSize {
			return buf.Bytes(), "jpeg", nil
		}
	}

	return nil, "", errors.New("image can not be made to fit within max size")
}

func downloadImage(url string) ([]byte, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	// Verify that the image is valid
	_, format, err := image.Decode(bytes.NewReader(body))
	return body, format, err
}
