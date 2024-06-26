package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"common"

	"github.com/gorilla/mux"
	"golang.org/x/sync/errgroup"
)

var apiKey string

func resolveVanityURL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url := fmt.Sprintf("http://api.steampowered.com/ISteamUser/ResolveVanityURL/v0001/?key=%s&vanityurl=%s", apiKey, vars["login"])
	log.Println(url)
	var vanityResp common.VanityURLResponse
	common.FetchJson(url, &vanityResp)
	json.NewEncoder(w).Encode(vanityResp)
}

func getRecentGames(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	steamID := vars["steamID"]

	url := fmt.Sprintf("http://api.steampowered.com/IPlayerService/GetRecentlyPlayedGames/v1/?key=%s&steamid=%s&format=json&count=5", apiKey, steamID)
	log.Println(url)
	var recentGamesResponse common.RecentGamesResponse
	if err := common.FetchJson(url, &recentGamesResponse); err != nil {
		http.Error(w, "Failed to fetch recent games", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(recentGamesResponse)
}

func getGlobalAchievementPercentagesForApp(appID string) (map[string]float64, error) {
	url := fmt.Sprintf("https://api.steampowered.com/ISteamUserStats/GetGlobalAchievementPercentagesForApp/v2/?key=%s&gameid=%s", apiKey, appID)
	log.Println(url)
	var globalAchResponse struct {
		AchievementPercentages struct {
			Achievements []struct {
				Name    string  `json:"name"`
				Percent float64 `json:"percent"`
			} `json:"achievements"`
		} `json:"achievementpercentages"`
	}
	if err := common.FetchJson(url, &globalAchResponse); err != nil {
		return nil, err
	}
	res := make(map[string]float64)
	for _, ach := range globalAchResponse.AchievementPercentages.Achievements {
		res[ach.Name] = ach.Percent
	}
	return res, nil
}

func getIcons(appID string) (map[string]string, error) {
	url := fmt.Sprintf("https://api.steampowered.com/ISteamUserStats/GetSchemaForGame/v2/?key=%s&appid=%s&l=english", apiKey, appID)
	log.Println(url)
	var iconsResponse struct {
		Game struct {
			Info struct {
				Achievements []struct {
					Apiname string `json:"name"`
					Icon    string `json:"icon"`
				} `json:"achievements"`
			} `json:"availableGameStats"`
		} `json:"game"`
	}
	if err := common.FetchJson(url, &iconsResponse); err != nil {
		return nil, err
	}
	res := make(map[string]string)
	for _, ach := range iconsResponse.Game.Info.Achievements {
		res[ach.Apiname] = ach.Icon
	}
	return res, nil
}

func getAchievementsInfo(w http.ResponseWriter, r *http.Request) {
	g, _ := errgroup.WithContext(context.Background())
	vars := mux.Vars(r)
	appID := vars["appID"]
	steamID := vars["steamID"]

	url := fmt.Sprintf("http://api.steampowered.com/ISteamUserStats/GetPlayerAchievements/v1/?key=%s&steamid=%s&appid=%s&l=english", apiKey, steamID, appID)
	log.Println(url)
	var achResponse common.AchievementResponse
	g.Go(func() error {
		return common.FetchJson(url, &achResponse)
	})

	var percentages map[string]float64
	g.Go(func() error {
		var err error
		percentages, err = getGlobalAchievementPercentagesForApp(appID)
		return err
	})

	var icons map[string]string
	g.Go(func() error {
		var err error
		icons, err = getIcons(appID)
		return err
	})

	if err := g.Wait(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for i, ach := range achResponse.Playerstats.Achievements {
		perc, ok := percentages[ach.Apiname]
		if ok {
			achResponse.Playerstats.Achievements[i].Percent = perc
		}
		icon, ok := icons[ach.Apiname]
		if ok {
			achResponse.Playerstats.Achievements[i].Icon = icon
		}
	}
	json.NewEncoder(w).Encode(achResponse)
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	var response common.HealthCheckResponse
	response.Alive = true
	json.NewEncoder(w).Encode(response)
}

func main() {
	data, err := os.ReadFile("api_key")
	if err != nil {
		log.Fatal("Failed to read API key from file: ", err)
	}
	apiKey = string(data)

	if apiKey == "" {
		log.Fatal("API Key is required. Provide via -apiKey=<YOUR_STEAM_API_KEY> or in api_key file")
	}

	r := mux.NewRouter()
	r.HandleFunc("/status", HealthCheck).Methods("GET")
	r.HandleFunc("/login-to-id/{login}", resolveVanityURL).Methods("GET")
	r.HandleFunc("/user/{steamID}/recent-games", getRecentGames).Methods("GET")
	r.HandleFunc("/user/{steamID}/achievements/{appID}", getAchievementsInfo).Methods("GET")
	log.Fatal(http.ListenAndServe(":8000", r))
}
