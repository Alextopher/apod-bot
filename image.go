package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"runtime"

	_ "image/png"

	"golang.org/x/image/draw"
)

// MAX_IMAGE_SIZE is 8MB
const MAX_IMAGE_SIZE = 8 * 1024 * 1024

// resizeImage reads an image and resizes it to be close to max_size
func resizeImage(img []byte, max_size int) ([]byte, error) {
	// Decode the image
	m, format, err := image.Decode(bytes.NewReader(img))
	if err != nil {
		log.Println("Can not decode", format, "images")
		return nil, err
	}

	// Downsample the image many times and pick the best one
	width := m.Bounds().Dx()

	// Choose 10 widths to try
	widths := []int{(width * 9) / 10, (width * 8) / 10, (width * 7) / 10, (width * 6) / 10, (width * 5) / 10, (width * 4) / 10, (width * 3) / 10, (width * 2) / 10, (width * 1) / 10, width}
	images := make(chan []byte, len(widths))

	// Start a goroutine for each width
	for _, w := range widths {
		// Calculate the height of the image
		height := int(m.Bounds().Dy() * w / m.Bounds().Dx())

		// Resize the image
		resized := image.NewRGBA(image.Rect(0, 0, w, height))
		draw.NearestNeighbor.Scale(resized, resized.Bounds(), m, m.Bounds(), draw.Over, nil)

		// Encode the image
		buf := new(bytes.Buffer)
		if err := jpeg.Encode(buf, resized, &jpeg.Options{
			Quality: 100,
		}); err != nil {
			log.Println("Could not encode image")
		} else {
			images <- buf.Bytes()
		}
	}

	// Find the largest image that is still smaller than max_size
	var best []byte
	for i := 0; i < len(widths); i++ {
		img := <-images
		if len(img) < max_size && len(img) > len(best) {
			best = img
		}
	}

	if best == nil {
		return nil, fmt.Errorf("could not resize to a suitable image")
	}

	return best, nil
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
	if len(bytes) > MAX_IMAGE_SIZE {
		img, err := resizeImage(bytes, MAX_IMAGE_SIZE)
		runtime.GC()
		if err != nil {
			return nil, err
		}

		return img, nil
	}

	return bytes, nil
}
