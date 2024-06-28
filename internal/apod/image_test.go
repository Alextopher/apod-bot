package apod

import (
	"testing"
	"time"
)

type Pair[T any, U any] struct {
	first  T
	second U
}

func NewPair[T, U any](first T, second U) Pair[T, U] {
	return Pair[T, U]{first, second}
}

const DiscordMaxImageSize = 8 * 1024 * 1024

func TestRegressions(t *testing.T) {
	urls := []Pair[string, string]{
		// Sep 26, 2023 - Image large image failed to resize properly
		NewPair("2023-09-26", "https://apod.nasa.gov/apod/image/2309/BlueHorse_Grelin_9342.jpg"),
	}

	for _, pair := range urls {
		start := time.Now()
		wrapper, err := downloadImage(pair.second)

		if err != nil {
			t.Error(err)
		}

		t.Log("Downloaded image in", time.Since(start))
		start = time.Now()

		// Resize the image
		err = wrapper.Resize(DiscordMaxImageSize)
		if err != nil {
			t.Error(err)
		}

		t.Log("Resized image in", time.Since(start))
	}
}
