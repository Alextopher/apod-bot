package main

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"runtime"

	_ "image/gif"
	_ "image/png"
)

// MaxImageSize is 8MB
const MaxImageSize = 8 * 1024 * 1024

// resizeImage reads an image and resizes it to be close to max_size
func resizeImage(img []byte, maxSize int) ([]byte, error) {
	// Decode the image
	message, format, err := image.Decode(bytes.NewReader(img))
	if err != nil {
		log.Println("Can not decode a '", format, "' image")
		return nil, err
	}

	buf := new(bytes.Buffer)

	// If the image is already small enough, return it
	if len(img) < maxSize {
		return img, nil
	}

	// Decrease the quality of the image until it fits within the max size
	for quality := 100; quality > 0; quality -= 5 {
		err := jpeg.Encode(buf, message, &jpeg.Options{
			Quality: quality,
		})

		if err != nil {
			log.Println("Could not encode image")
		}

		if buf.Len() < maxSize {
			return buf.Bytes(), nil
		}
	}

	return nil, errors.New("image can not be made to fit within max size")
}

func downloadImage(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// If the image is too large reformat it to a smaller size
	if len(bytes) > MaxImageSize {
		img, err := resizeImage(bytes, MaxImageSize)
		runtime.GC()
		if err != nil {
			return nil, err
		}

		return img, nil
	}

	return bytes, nil
}
