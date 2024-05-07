package album

type Album struct {
	Name    string   `json:"name"`
	Artists []Artist `json:"artists"`
}

type Artist struct {
	Name string `json:"name"`
}

type SavedAlbum struct {
	Album Album `json:"album"`
}

type UserSavedAlbums struct {
	Items []SavedAlbum `json:"items"`
}
