package spotify

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/jonathanbell/spotify2ytmusic/api/spotify"
	"github.com/jonathanbell/spotify2ytmusic/pkg/playlist"
	"github.com/jonathanbell/spotify2ytmusic/pkg/track"
)

type Spotify struct {
	oauthResponse spotify.OAuthResponse
	mutex         *sync.Mutex
	tokenReady    chan struct{}
	tokenExpires  time.Time
}

func (s *Spotify) Authenticate() {
	s.mutex = &sync.Mutex{}
	// Initialize the channel.
	s.tokenReady = make(chan struct{})

	http.HandleFunc("/redirect", s.redirectHandler)
	http.HandleFunc("/capture", s.captureHandler)

	// Start a local server to parse the OAuth response.
	go func() {
		if err := http.ListenAndServe(":"+spotify.ResponsePort, nil); err != nil {
			fmt.Println("Failed to start server: ", err)
		}
	}()

	var oauthUrl string = "https://accounts.spotify.com/authorize?response_type=token&client_id=5c098bcc800e45d49e476265bc9b6934&scope=playlist-read-private%20playlist-read-collaborative%20user-library-read&redirect_uri=http://127.0.0.1:" + spotify.ResponsePort + "/redirect"

	openBrowser(oauthUrl)

	// Wait for the OAuth response.
	select {
	case <-s.tokenReady: // Block until the token is ready
	case <-time.After(120 * time.Second): // Timeout after 120 seconds
		fmt.Println("Failed to get OAuth response within 120 seconds")
		return
	}
}

func (s *Spotify) DisplayUserInfo() {
	req, err := http.NewRequest("GET", spotify.BaseApiUrl+"/me", nil)
	if err != nil {
		fmt.Println("Failed to create GetUserInfo request: ", err)
		return
	}

	req.Header.Add("Authorization", "Bearer "+s.oauthResponse.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Failed to make DefaultClient request: ", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read GetUserInfo response: ", err)
		return
	}

	var userInfo struct {
		Id          string `json:"id"`
		DisplayName string `json:"display_name"`
	}

	err = json.Unmarshal(body, &userInfo)
	if err != nil {
		fmt.Println("Failed to parse GetUserInfo JSON response: ", err)
		return
	}

	fmt.Println("User logged in as: ", userInfo.DisplayName)
	fmt.Println("Using Bearer token: ", s.oauthResponse.AccessToken)
}

func (s *Spotify) GetUserPlaylists() map[string][]track.Track {
	req, err := http.NewRequest("GET", spotify.BaseApiUrl+"/me/playlists", nil)
	if err != nil {
		fmt.Println("Failed to create GetUserPlaylists request: ", err)
		return nil
	}

	req.Header.Add("Authorization", "Bearer "+s.oauthResponse.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Failed to make DefaultClient request: ", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read GetUserPlaylists response: ", err)
		return nil
	}

	var userPlaylists playlist.UserPlaylists

	err = json.Unmarshal(body, &userPlaylists)
	if err != nil {
		fmt.Println("Failed to parse GetUserPlaylists JSON response: ", err)
		return nil
	}

	playlists := make(map[string][]track.Track)

	for _, playlist := range userPlaylists.Items {
		fmt.Printf("🎧 Playlist: %s\n", playlist.Name)
		tracks := s.GetPlaylistTracks(playlist.ID)
		playlists[playlist.ID] = tracks
	}

	return playlists
}

func (s *Spotify) GetPlaylistTracks(playlistID string) []track.Track {
	req, err := http.NewRequest("GET", spotify.BaseApiUrl+"/playlists/"+playlistID+"/tracks", nil)
	if err != nil {
		fmt.Println("Failed to create GetPlaylistTracks request: ", err)
		return nil
	}

	req.Header.Add("Authorization", "Bearer "+s.oauthResponse.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Failed to make DefaultClient request: ", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read GetPlaylistTracks response: ", err)
		return nil
	}

	var playlistTracks playlist.PlaylistTracks

	err = json.Unmarshal(body, &playlistTracks)
	if err != nil {
		fmt.Println("Failed to parse GetPlaylistTracks JSON response: ", err)
		return nil
	}

	var tracks []track.Track
	for _, item := range playlistTracks.Items {
		// Print the name of the first artist. Note that a song can have multiple artists.
		if len(item.Track.Artists) > 0 {
			fmt.Printf("🎶 Track: %s - %s (%s)\n", item.Track.Artists[0].Name, item.Track.Title, item.Track.Album.Name)
			tracks = append(tracks, track.Track{Title: item.Track.Title, Artists: item.Track.Artists, Album: item.Track.Album})
		} else {
			fmt.Printf("🎶 Track: Unknown Artist - %s (%s)\n", item.Track.Title, item.Track.Album.Name)
			tracks = append(tracks, track.Track{Title: item.Track.Title, Artists: nil, Album: item.Track.Album})
		}
	}

	return tracks
}

// func (s *Spotify) GetLikedSongs() []track.Track {
// 	req, err := http.NewRequest("GET", spotify.BaseApiUrl+"/me/tracks", nil) {
// 	if err != nil {
// 		fmt.Println("Failed to create GetLikedSongs request: ", err)
// 		return nil
// 	}

// 	req.Header.Add("Authorization", "Bearer "+s.oauthResponse.AccessToken)

// 	resp, err := http.DefaultClient.Do(req)
// 	if err != nil {
// 		fmt.Println("Failed to make DefaultClient request: ", err)
// 		return nil
// 	}

// 	defer resp.Body.Close()

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		fmt.Println("Failed to read GetLikedSongs response: ", err)
// 		return nil
// 	}

// 	var likedSongs playlist.PlaylistTracks

// }

// func newSpotifyAuth() *Spotify {
// 	return &Spotify{
// 		mutex: &sync.Mutex{},
// 		// Initialize the channel.
// 		tokenReady: make(chan struct{}),
// 	}
// }

func (s *Spotify) redirectHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `
		<html>
		<head>
			<script>
				window.onload = function() {
					const hash = window.location.hash.substring(1);
					const params = new URLSearchParams(hash);
					window.location.href = "/capture?" + params.toString();
				}
			</script>
		</head>
		<body>
			Redirecting...
		</body>
		</html>
	`)
}

func (s *Spotify) captureHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	if error := query.Get("error"); error != "" {
		fmt.Println("Error during OAuth authorization: ", error)
		fmt.Fprintf(w, "<html><body>🚨 An error occurred during OAuth authorization: %s.</body></html>", error)
		return
	}

	response := spotify.OAuthResponse{
		AccessToken: query.Get("access_token"),
		TokenType:   query.Get("token_type"),
		ExpiresIn:   query.Get("expires_in"),
	}

	expiresIn := 3600
	if response.ExpiresIn != "" {
		expiresIn, _ = strconv.Atoi(response.ExpiresIn)
	}

	s.mutex.Lock()
	s.oauthResponse = response
	// Calculate the time when the token will expire.
	s.tokenExpires = time.Now().Add(time.Duration(expiresIn) * time.Second)
	s.mutex.Unlock()

	// Close the channel to signal that the token is ready.
	close(s.tokenReady)

	fmt.Fprintf(w, "<html><body>✅ OAuth authorization complete. You may close this tab.</body></html>")
}

func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		fmt.Println("Failed to open browser:", err)
	}
}
