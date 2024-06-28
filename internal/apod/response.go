package apod

import (
	"fmt"
)

// Response is a single JSON response from the APOD API.
type Response struct {
	Title       string `json:"title"`
	Date        string `json:"date"`
	Url         string `json:"url"`
	HdUrl       string `json:"hdurl"`
	MediaType   string `json:"media_type"`
	Explanation string `json:"explanation"`
	Thumbnail   string `json:"thumbnail_url"`
	Copyright   string `json:"copyright"`
	Service     string `json:"service_version"`
}

func (a *Response) String() string {
	return fmt.Sprintf("Title: %s\nDate: %s\nURL: %s\nHDURL: %s\nMediaType: %s\nExplanation: %s\nThumbnail: %s\nCopyright: %s\nService: %s", a.Title, a.Date, a.Url, a.HdUrl, a.MediaType, a.Explanation, a.Thumbnail, a.Copyright, a.Service)
}

// CreateExplanation creates a markdown formatted explanation of today's APOD
func (a *Response) CreateExplanation() string {
	return fmt.Sprintf("_%s_\n> %s", a.Title, a.Explanation)
}

// DownloadRawImage downloads the image without resizing
func (a *Response) DownloadRawImage() (*ImageWrapper, error) {
	if a.MediaType == "image" {
		return downloadImage(a.HdUrl)
	}
	return downloadImage(a.Thumbnail)
}

func (a *Response) GetDate() string {
	return a.Date
}
