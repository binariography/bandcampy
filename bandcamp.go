package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gocolly/colly"
)

var embedAlbumArt bool

// Define a function to handle command-line flags
func init() {
	// Register flags for embedding album art
	flag.BoolVar(&embedAlbumArt, "embed-album-art", false, "Embed the album art into tracks?[default:false]")
	flag.Parse()
}

// Config struct for environment and file paths
type Config struct {
	UserHomeDir string
	BaseDir     string
}

// File represents the file details in JSON
type File struct {
	MP3_128 string `json:"mp3-128"`
}

// TrackInfo represents a single track's data
type TrackInfo struct {
	ID        *int    `json:"id"`
	TrackID   int     `json:"track_id"`
	File      *File   `json:"file"`
	Artist    *string `json:"artist"`
	Title     string  `json:"title"`
	TrackNum  int     `json:"track_num"`
	Duration  float64 `json:"duration"`
	AltLink   *string `json:"alt_link"`
	PlayCount *int    `json:"play_count"`
	IsCapped  *bool   `json:"is_capped"`
}

// Album represents an album with track information and cover image link
type Album struct {
	Name           string
	TrackInfo      []TrackInfo `json:"trackinfo"`
	CoverImageLink string
	Artist         string `json:"Artist"`
}

var site string

func init() {
	// Use command-line argument to form the Bandcamp URL
	if len(os.Args) < 2 {
		log.Fatal("Please provide the Bandcamp URL subdomain as a command-line argument.")
	}
	site = "https://" + os.Args[1] + ".bandcamp.com"
}

func main() {
	// Setup config and fetch album details
	var config Config
	config.setEnvVars()

	// Handle Bandcamp URL input
	var Link string
	myAlbum := Album{}
	if strings.Contains(os.Args[1], "album") {
		Link = os.Args[1]
	} else {
		log.Fatal("No album is provided")
	}

	// Create a new collector for web scraping
	c := colly.NewCollector()

	// Extract album data from the page
	c.OnHTML("script[data-tralbum]", func(e *colly.HTMLElement) {
		link := e.Attr("data-tralbum")
		myAlbum.setVars(link)
	})

	// Extract cover image URL from meta tag
	c.OnHTML("meta[property]", func(e *colly.HTMLElement) {
		if e.Attr("property") == "og:image" {
			myAlbum.CoverImageLink = e.Attr("content")
		}
	})

	// Start scraping the album page
	err := c.Visit(Link)
	if err != nil {
		log.Fatal("Error visiting the page: ", err)
	}

	// Fetch and download album tracks
	myAlbum.getTracks(&config)
	fmt.Println("Cover Image Link:", myAlbum.CoverImageLink)
}

// Set environment variables for the config
func (c *Config) setEnvVars() {
	c.UserHomeDir, _ = os.UserHomeDir()
}

// Set album variables from JSON data
func (A *Album) setVars(jsonData string) {
	err := json.Unmarshal([]byte(jsonData), A)
	if err != nil {
		log.Fatalf("Failed to parse album JSON: %v", err)
	}
}

// Download tracks and save them to the specified destination
func (A *Album) getTracks(C *Config) {
	artist := strings.ToLower(A.Artist)
	album := strings.ToLower(A.Name)
	baseDir := fmt.Sprintf("%v/bandcamp/", C.UserHomeDir)
	if C.BaseDir != "" {
		baseDir = fmt.Sprintf("%v/", C.BaseDir)
	}
	destDir := fmt.Sprintf("%v%v/%v/", baseDir, artist, album)

	// Create destination directory if it doesn't exist
	if err := createDirIfNotExists(destDir); err != nil {
		log.Fatal("Failed to create directory: ", err)
	}

	// Download each track
	for _, v := range A.TrackInfo {
		if v.File != nil {
			fileName := removeIllegalCharacters(fmt.Sprintf("%d-%v.mp3", v.TrackNum, v.Title))
			downloadTracks(v.File.MP3_128, fileName, destDir)
		}
	}
}

// Create a directory if it doesn't already exist
func createDirIfNotExists(destDir string) error {
	_, err := os.Stat(destDir)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(destDir, 0755)
		if errDir != nil {
			return fmt.Errorf("failed to create directory: %v", errDir)
		}
	}
	return nil
}

// Download a track from the provided URL
func downloadTracks(url, fileName, destDir string) {
	filepath := fmt.Sprintf("%v/%v", destDir, fileName)

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		log.Printf("Failed to create file %v: %v", filepath, err)
		return
	}
	defer out.Close()

	// Get the data from the URL
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to download track %v: %v", url, err)
		return
	}
	defer resp.Body.Close()

	// Write the body to the file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Printf("Failed to write to file %v: %v", filepath, err)
	}
}

// Remove illegal characters from filenames
func removeIllegalCharacters(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' {
			log.Printf("Removed %c from filename: %v\n", r, s)
			return -1
		}
		return r
	}, s)
}
