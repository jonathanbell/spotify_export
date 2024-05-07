package spotify

const ResponsePort = "43019"
const ClientID = ""
const BaseApiUrl = "https://api.spotify.com/v1"

type OAuthResponse struct {
	AccessToken string
	TokenType   string
	ExpiresIn   string
}
