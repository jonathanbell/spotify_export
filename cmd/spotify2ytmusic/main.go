package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jonathanbell/spotify2ytmusic/pkg/spotify"
)

func main() {
	// Instantiate Spotify struct.
	spotifyInstance := &spotify.Spotify{}

	// Authenticate with Spotify.
	spotifyInstance.Authenticate()

	// Get the user's information.
	spotifyInstance.DisplayUserInfo()

	playlists := spotifyInstance.GetUserPlaylists()

	data, err := json.MarshalIndent(playlists, "", "  ")
	if err != nil {
		fmt.Println("Failed to serialize playlists: ", err)
		return
	}

	err = os.WriteFile("data/spotify/spotify_library.json", data, 0644)
	if err != nil {
		fmt.Println("Failed to write Spotify playlists to file: ", err)
		return
	}

	fmt.Println("✅ Spotify library exported successfully.")
}
