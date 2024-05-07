package playlist

import (
	"github.com/jonathanbell/spotify_export/pkg/track"
)

type Playlist struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	TrackCount int           `json:"track_count"`
	Tracks     []track.Track `json:"tracks"`
}

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
