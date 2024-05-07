package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jonathanbell/spotify_export/pkg/spotify"
)

func main() {
	spotifyInstance := &spotify.Spotify{}

	// Authenticate with Spotify.
	spotifyInstance.Authenticate()

	// Get the user's information.
	spotifyInstance.DisplayUserInfo()

	// Do the do.
	library := spotifyInstance.ExportUserLibrary()

	data, err := json.MarshalIndent(library, "", "  ")
	if err != nil {
		fmt.Println("Failed to serialize Spotify library as JSON: ", err)
		return
	}

	desktopPath, _ := os.UserHomeDir()
	desktopFilePath := filepath.Join(desktopPath, "Desktop", "spotify_library.json")
	err = os.WriteFile(desktopFilePath, data, 0644)
	if err != nil {
		fmt.Println("Failed to write Spotify playlists to file: ", err)
		return
	}

	fmt.Println("✅ Spotify library (spotify_library.json) exported successfully to the desktop.")
}
