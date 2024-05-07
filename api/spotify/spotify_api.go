package spotify

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const ResponsePort = "43019"
const ClientID = ""
const BaseApiUrl = "https://api.spotify.com/v1"

type OAuthResponse struct {
	AccessToken string
	TokenType   string
	ExpiresIn   string
}

func Get(path string, token string, params map[string]string) []byte {
	var allItems []interface{}
	nextPath := BaseApiUrl + path

	for nextPath != "" {
		req, err := http.NewRequest("GET", nextPath, nil)
		if err != nil {
			fmt.Println("Failed to create GET request for path ("+nextPath+"): ", err)
			return nil
		}

		// Add query strings to the first page/request.
		if nextPath == BaseApiUrl+path {
			query := req.URL.Query()
			for key, value := range params {
				query.Add(key, value)
			}
			req.URL.RawQuery = query.Encode()
		}

		req.Header.Add("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("Failed to make DefaultClient request: ", err)
			return nil
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			fmt.Println("Failed to GET path ("+nextPath+"): ", resp.Status)
			return nil
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Failed to read body response: ", err)
			return nil
		}

		var pageData map[string]interface{}
		if err := json.Unmarshal(body, &pageData); err != nil {
			fmt.Println("Failed to unmarshal JSON response: ", err)
			return nil
		}

		// Extract items and append to allItems.
		if items, found := pageData["items"].([]interface{}); found {
			allItems = append(allItems, items...)
		}

		// Check for the next page URL.
		if next, ok := pageData["next"].(string); ok && next != "" {
			nextPath = next
		} else {
			break
		}
	}

	result := map[string]interface{}{
		"items": allItems,
	}

	finalData, err := json.Marshal(result)
	if err != nil {
		fmt.Println("Failed to marshal final JSON data: ", err)
		return nil
	}

	return finalData
}
