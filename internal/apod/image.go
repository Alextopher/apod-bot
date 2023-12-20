package apod

import (
	"bytes"
	"image"
	"io"
	"log"
	"net/http"

	"image/jpeg"

	// Import additional image formats
	_ "image/gif"
	_ "image/png"
)

// DiscordMaxImageSize is the maximum size (in bytes) of an image that can be sent to Discord
const DiscordMaxImageSize = 8 * 1024 * 1024

// ImageWrapper is a wrapper around an image.Image that caches the image's
// binary representation.
type ImageWrapper struct {
	// Image is the decoded image
	Image image.Image
	// Format is the image format (png, jpg, etc.)
	Format string
	// Bytes is the binary representation of the image
	Bytes []byte
}

// NewImageWrapper creates a new ImageWrapper from binary data.
func NewImageWrapper(buf []byte) (*ImageWrapper, error) {
	img, format, err := image.Decode(bytes.NewReader(buf))
	return &ImageWrapper{
		Image:  img,
		Format: format,
		Bytes:  buf,
	}, err
}

// Resize modifies the image to be under the max size.
func (i *ImageWrapper) Resize(maxBytes int) error {
	if len(i.Bytes) <= maxBytes {
		return nil
	}

	// Re encode the image with lower quality until it is under the max size
	for quality := 100; quality > 0; quality -= 5 {
		buf := &bytes.Buffer{}
		err := jpeg.Encode(buf, i.Image, &jpeg.Options{Quality: quality})
		if err != nil {
			return err
		}

		if buf.Len() <= maxBytes {
			i.Bytes = buf.Bytes()
			return nil
		}
	}

	return nil
}

func downloadImage(url string) (*ImageWrapper, error) {
	log.Println("Downloading image from", url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return NewImageWrapper(body)
}
