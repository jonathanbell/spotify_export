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

	spotify_api "github.com/jonathanbell/spotify_export/api/spotify"
	"github.com/jonathanbell/spotify_export/pkg/album"
	"github.com/jonathanbell/spotify_export/pkg/artist"
	"github.com/jonathanbell/spotify_export/pkg/playlist"
	"github.com/jonathanbell/spotify_export/pkg/track"
)

type Spotify struct {
	oauthResponse spotify_api.OAuthResponse
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
		if err := http.ListenAndServe(":"+spotify_api.ResponsePort, nil); err != nil {
			fmt.Println("Failed to start server: ", err)
		}
	}()

	var oauthUrl string = "https://accounts.spotify.com/authorize?response_type=token&client_id=5c098bcc800e45d49e476265bc9b6934&scope=playlist-read-private%20playlist-read-collaborative%20user-library-read%20user-follow-read&redirect_uri=http://127.0.0.1:" + spotify_api.ResponsePort + "/redirect"

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
	var userInfo struct {
		Id          string `json:"id"`
		DisplayName string `json:"display_name"`
	}

	req, err := http.NewRequest("GET", spotify_api.BaseApiUrl+"/me", nil)
	if err != nil {
		fmt.Println("Failed to create GET request user info endpoint: ", err)
		return
	}

	req.Header.Add("Authorization", "Bearer "+s.oauthResponse.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Failed to send request to user info endpoint: ", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read response body: ", err)
		return
	}

	err = json.Unmarshal(body, &userInfo)
	if err != nil {
		fmt.Println("Failed to parse GetUserInfo JSON response: ", err)
		return
	}

	fmt.Println("User logged in as: ", userInfo.DisplayName)
	fmt.Println("Using Bearer token: ", s.oauthResponse.AccessToken)
}

func (s *Spotify) GetUserLikedSongs() []playlist.Playlist {
	params := map[string]string{
		"limit": "50",
	}
	body := spotify_api.Get("/me/tracks", s.oauthResponse.AccessToken, params)

	var likedSongs playlist.PlaylistTracks

	err := json.Unmarshal(body, &likedSongs)
	if err != nil {
		fmt.Println("Failed to parse GetUserLikedSongs JSON response: ", err)
		return nil
	}

	tracks := []track.Track{}

	for _, item := range likedSongs.Items {
		// Print the name of the first artist. Note that a song can have multiple artists.
		if len(item.Track.Artists) > 0 {
			fmt.Printf("🎶 Track: %s - %s (%s)\n", item.Track.Artists[0].Name, item.Track.Title, item.Track.Album.Name)
			tracks = append(tracks, track.Track{Title: item.Track.Title, Artists: item.Track.Artists, Album: item.Track.Album})
		} else {
			fmt.Printf("🎶 Track: Unknown Artist - %s (%s)\n", item.Track.Title, item.Track.Album.Name)
			tracks = append(tracks, track.Track{Title: item.Track.Title, Artists: nil, Album: item.Track.Album})
		}
	}

	fmt.Printf("🥰 Liked Songs (%d tracks)\n\n", len(tracks))

	p := playlist.Playlist{
		ID:         "1",
		Name:       "Liked Songs",
		TrackCount: len(tracks),
		Tracks:     tracks,
	}

	playlists := []playlist.Playlist{p}
	return playlists
}

func (s *Spotify) GetUserPlaylists() []playlist.Playlist {
	body := spotify_api.Get("/me/playlists", s.oauthResponse.AccessToken, nil)

	var userPlaylists playlist.UserPlaylists

	err := json.Unmarshal(body, &userPlaylists)
	if err != nil {
		fmt.Println("Failed to parse GetUserPlaylists JSON response: ", err)
		return nil
	}

	playlists := []playlist.Playlist{}

	for _, plist := range userPlaylists.Items {
		tracks := s.getPlaylistTracks(plist.ID)
		p := playlist.Playlist{
			ID:         plist.ID,
			Name:       plist.Name,
			TrackCount: len(tracks),
			Tracks:     tracks,
		}

		playlistInfo := fmt.Sprintf("🎧 Playlist: %s (%d tracks)\n", p.Name, len(p.Tracks))
		fmt.Println(playlistInfo)
		playlists = append(playlists, p)
	}

	return playlists
}

func (s *Spotify) GetUserFollowedArtists() []artist.Artist {
	var res struct {
		Artists struct {
			Items []artist.Artist `json:"items"`
		} `json:"artists"`
	}

	req, err := http.NewRequest("GET", spotify_api.BaseApiUrl+"/me/following?type=artist&limit=50", nil)
	if err != nil {
		fmt.Println("Failed to create GET request for GetUserFollowedArtists: ", err)
		return nil
	}

	req.Header.Add("Authorization", "Bearer "+s.oauthResponse.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Failed to send request to user following endpoint: ", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read response body (GetUserFollowedArtists): ", err)
		return nil
	}

	err = json.Unmarshal(body, &res)
	if err != nil {
		fmt.Println("Failed to unmarshal JSON response for followed artists: ", err)
		return nil
	}

	followedArtists := append([]artist.Artist{}, res.Artists.Items...)

	return followedArtists
}

func (s *Spotify) GetUserSavedAlbums() []album.Album {
	body := spotify_api.Get("/me/albums", s.oauthResponse.AccessToken, map[string]string{"limit": "50"})

	var userSavedAlbums album.UserSavedAlbums

	err := json.Unmarshal(body, &userSavedAlbums)
	if err != nil {
		fmt.Println("Failed to parse GetUserSavedAlbums JSON response: ", err)
		return nil
	}

	albums := make([]album.Album, 0, len(userSavedAlbums.Items))

	for _, savedAlbum := range userSavedAlbums.Items {
		fmt.Printf("📀 Album: %s - %s\n", savedAlbum.Album.Artists[0].Name, savedAlbum.Album.Name)
		albums = append(albums, savedAlbum.Album)
	}

	fmt.Printf("📀 Saved Albums (%d)\n\n", len(albums))

	return albums
}

func (s *Spotify) ExportUserLibrary() map[string]interface{} {
	likedSongs := s.GetUserLikedSongs()
	playlists := s.GetUserPlaylists()
	followedArtists := s.GetUserFollowedArtists()
	savedAlbums := s.GetUserSavedAlbums()

	userLibrary := map[string]interface{}{
		"liked_songs":      likedSongs,
		"playlists":        playlists,
		"saved_albums":     savedAlbums,
		"followed_artists": followedArtists,
	}
	return userLibrary
}

func (s *Spotify) getPlaylistTracks(playlistID string) []track.Track {
	body := spotify_api.Get("/playlists/"+playlistID+"/tracks", s.oauthResponse.AccessToken, nil)

	var playlistTracks playlist.PlaylistTracks

	err := json.Unmarshal(body, &playlistTracks)
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

	response := spotify_api.OAuthResponse{
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

	fmt.Fprintf(w, "<html><body>✅ OAuth authorization complete. Spotify library export will begin shortly. Check your desktop for <code>spotify_library.json</code><br>You may close this tab. 👋</body></html>")
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
