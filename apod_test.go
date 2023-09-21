package main

// Verify that api commands are working as expected
import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

func CreateAPOD() *APOD {
	key, ok := os.LookupEnv("APOD_TOKEN")
	if !ok {
		godotenv.Load()
		key = os.Getenv("APOD_TOKEN")
	}

	// Empty cache (reads and writes nothing)
	cache, _ := NewAPODCache(strings.NewReader(""), io.Discard)

	// Empty image cache (reads and writes nothing)
	return &APOD{
		key:        key,
		cache:      cache,
		imageCache: NewImageCache(),
	}
}

// Verify that the default APOD request works
func TestToday(t *testing.T) {
	apod := CreateAPOD()

	_, err := apod.Today()
	if err != nil {
		t.Error(err)
	}
}

// Verify a particular date
func TestGet(t *testing.T) {
	apod := CreateAPOD()

	resp, err := apod.Get("2021-07-01")
	if err != nil {
		t.Error(err)
	}

	if resp.Title != "Perseverance Selfie with Ingenuity" {
		t.Error("Incorrect title")
	}

	if resp.Date != "2021-07-01" {
		t.Error("Incorrect date")
	}

	if resp.URL != "https://apod.nasa.gov/apod/image/2107/PIA24542_fig2_1100c.jpg" {
		t.Error("Incorrect URL")
	}

	if resp.HDURL != "https://apod.nasa.gov/apod/image/2107/PIA24542_fig2.jpg" {
		t.Error("Incorrect HDURL")
	}

	if resp.MediaType != "image" {
		t.Error("Incorrect MediaType")
	}

	if resp.Explanation != "On sol 46 (April 6, 2021) the Perseverance rover held out a robotic arm to take its first selfie on Mars. The WATSON camera at the end of the arm was designed to take close-ups of martian rocks and surface details though, and not a quick snap shot of friends and smiling faces. In the end, teamwork and weeks of planning on Mars time was required to program a complex series of exposures and camera motions to include Perseverance and its surroundings. The resulting 62 frames were composed into a detailed mosiac, one of the most complicated Mars rover selfies ever taken. In this version of the selfie, the rover's Mastcam-Z and SuperCam instruments are looking toward WATSON and the end of the rover's outstretched arm. About 4 meters (13 feet) from Perseverance is a robotic companion, the Mars Ingenuity helicopter." {
		t.Error("Incorrect explanation")
	}

	if resp.Thumbnail != "" {
		t.Error("Incorrect thumbnail")
	}

	if resp.Service != "v1" {
		t.Error("Incorrect service")
	}
}
