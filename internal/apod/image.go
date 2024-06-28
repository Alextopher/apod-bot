package apod

import (
	"bytes"
	"image"
	"io"
	"net/http"

	// Include jpeg image decoder
	"image/jpeg"
	// Include gif image decoder
	_ "image/gif"
	// Include png image decoder
	_ "image/png"

	"github.com/Alextopher/apod-bot/internal/cache"
)

// NewImageCache simplifies the creation of an APOD image cache
func NewImageCache(dir string) *cache.FSCache[*ImageWrapper] {
	return cache.NewFSCache(
		cache.NewLocalFS(dir),
		func(b []byte) (*ImageWrapper, error) {
			return NewImageWrapper(b)
		},
		func(iw *ImageWrapper) ([]byte, error) {
			return iw.Bytes, nil
		},
	)
}

// ImageWrapper is a wrapper around an image.Image that caches the image's
// binary representation and format.
//
// This is useful for caching images in memory
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

// downloadImage creates a new ImageWrapper from an image URL.
func downloadImage(url string) (*ImageWrapper, error) {
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

// Resize returns a new ImageWrapper with the original image converted to jpeg
// and reduces quality until the image is under maxBytes.
func (i *ImageWrapper) Resize(maxBytes int) error {
	if len(i.Bytes) <= maxBytes {
		return nil
	}

	// re-encode the image with lower quality until it is under the max size
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
