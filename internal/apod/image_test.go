package apod

import (
	"testing"
)

type Pair[T any, U any] struct {
	first  T
	second U
}

func NewPair[T, U any](first T, second U) Pair[T, U] {
	return Pair[T, U]{first, second}
}

func TestRegressions(t *testing.T) {
	urls := []Pair[string, string]{
		// Sep 26, 2023 - Image large image failed to resize properly
		NewPair("2023-09-26", "https://apod.nasa.gov/apod/image/2309/BlueHorse_Grelin_9342.jpg"),
	}

	imageCache, err := NewDirectoryImageCache("test_images")
	if err != nil {
		t.Error(err)
	}

	for _, pair := range urls {
		// Download full size images and save them to disk
		wrapper, err := GetOrSetImage(imageCache, pair.first, func() (wrapper *ImageWrapper, err error) {
			return downloadImage(pair.second)
		})

		if err != nil {
			t.Error(err)
		}

		// Resize the image
		err = wrapper.Resize(DiscordMaxImageSize)
		if err != nil {
			t.Error(err)
		}
	}
}
