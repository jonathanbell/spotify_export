package playlist

import (
	"github.com/jonathanbell/spotify2ytmusic/pkg/track"
)

type SpotifyPlaylist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type UserPlaylists struct {
	Items []SpotifyPlaylist `json:"items"`
}

type PlaylistTracks struct {
	Items []struct {
		track.Track `json:"track"`
	} `json:"items"`
}
