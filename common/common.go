package common

import (
	"encoding/json"
	"io"
	"net/http"
)

type RecentGamesResponse struct {
	Response struct {
		Totalcount int `json:"total_count"`
		Games      []struct {
			AppID int    `json:"appid"`
			Name  string `json:"name"`
		} `json:"games"`
	} `json:"response"`
}

type AchievementResponse struct {
	Playerstats struct {
		SteamID      string `json:"steamID"`
		Gamename     string `json:"gameName"`
		Achievements []struct {
			Apiname     string  `json:"apiname"`
			Achieved    int     `json:"achieved"`
			Unlocktime  uint64  `json:"unlocktime"`
			Displayname string  `json:"name"`
			Description string  `json:"description"`
			Percent     float64 `json:"percent"`
			Icon        string  `json:"icon"`
		} `json:"achievements"`
	} `json:"playerstats"`
}

type VanityURLResponse struct {
	Response struct {
		SteamID string `json:"steamid"`
		Success int    `json:"success"`
		Message string `json:"message"`
	} `json:"response"`
}

type HealthCheckResponse struct {
	Alive bool `json:"alive"`
}

func FetchJson(url string, target interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}
