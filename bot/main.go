package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	common "common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	cmap "github.com/orcaman/concurrent-map/v2"
)

type User struct {
	SteamID string
	AppID   int
}

func trySend(bot *tgbotapi.BotAPI, msg tgbotapi.Chattable) {
	if _, err := bot.Send(msg); err != nil {
		log.Println(err)
	}
}

func getSteamID(login string) (string, error) {
	var response common.VanityURLResponse
	if err := common.FetchJson(fmt.Sprintf("http://steam_backend:8000/login-to-id/%v", login), &response); err != nil {
		return "", err
	}
	return response.Response.SteamID, nil
}

func getRecentGames(steamID string) (common.RecentGamesResponse, error) {
	var response common.RecentGamesResponse
	err := common.FetchJson(fmt.Sprintf("http://steam_backend:8000/user/%v/recent-games", steamID), &response)
	return response, err
}

func getAchievements(steamID string, appID int) (common.AchievementResponse, error) {
	var response common.AchievementResponse
	err := common.FetchJson(fmt.Sprintf("http://steam_backend:8000/user/%v/achievements/%v", steamID, appID), &response)
	return response, err
}

func achievementsToText(r common.AchievementResponse) string {
	sort.Slice(r.Playerstats.Achievements, func(i, j int) bool {
		return r.Playerstats.Achievements[i].Percent < r.Playerstats.Achievements[j].Percent
	})

	total := len(r.Playerstats.Achievements)
	if total == 0 {
		return fmt.Sprintf("Game: <b>%v</b>\n<br>There are no achievements in this game!", r.Playerstats.Gamename)
	}
	obtained_info := ""
	obtained_cnt := 0
	unobtained_cnt := 0
	unobtained_info := ""
	for _, a := range r.Playerstats.Achievements {
		if a.Achieved == 1 && obtained_cnt < 3 {
			obtained_cnt += 1
			obtained_info += fmt.Sprintf("<b>%v. %v</b> (<i>top %.2f%%</i>)\n<blockquote>%v</blockquote>\n\n", obtained_cnt, a.Displayname, a.Percent, a.Description)
		} else if a.Achieved == 0 && unobtained_cnt < 3 {
			unobtained_cnt += 1
			unobtained_info += fmt.Sprintf("<b>%v. %v</b> (<i>top %.2f%%</i>)\n<blockquote>%v</blockquote>\n\n", unobtained_cnt, a.Displayname, a.Percent, a.Description)
		}
	}

	res := fmt.Sprintf("\nGame: <b>%v</b>\n\n", r.Playerstats.Gamename)
	if unobtained_cnt > 0 {
		res += fmt.Sprintf("<b>Hardest unobtained:</b>\n\n%v\n\n", unobtained_info)
	} else {
		res += "All achievements collected! \U0001F3C6\n"
	}
	if obtained_cnt > 0 {
		res += fmt.Sprintf("<b>Top obtained:</b>\n\n%v\n\n", obtained_info)
	} else {
		res += "No obtained achievements yet!\n\n"
	}

	return res
}

func main() {
	telegram_token, err := os.ReadFile("telegram_bot_token")
	if err != nil {
		log.Fatal("Failed to read API key from file: ", err)
	}
	bot, err := tgbotapi.NewBotAPI(string(telegram_token))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false

	log.Printf("Authorized on account %v", bot.Self.UserName)

	sharding_function := func(key int64) uint32 {
		return uint32(key % int64(cmap.SHARD_COUNT))
	}
	users := cmap.NewWithCustomShardingFunction[int64, User](sharding_function)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if (update.Message == nil || !update.Message.IsCommand()) && update.CallbackQuery == nil {
			continue
		}

		go func(update tgbotapi.Update) {
			if update.Message != nil && update.Message.IsCommand() {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
				msg.ReplyToMessageID = update.Message.MessageID
				switch update.Message.Command() {
				case "help":
					msg.Text = `Available commands:
							/status – check backend status
							/register {steam_login} – set your steam login
							/recent_games – get recently played games
							/achievements – get all achievements for your recently played game`
					trySend(bot, msg)
				case "register":
					msg.Text = update.Message.Text
					words := strings.Split(update.Message.Text, " ")
					if len(words) != 2 {
						msg.Text = "Usage: /register {steam_login}"
						trySend(bot, msg)
						return
					}
					steamID, err := getSteamID(words[1])
					if err != nil {
						msg.Text = "Invalid login"
						trySend(bot, msg)
						return
					}
					users.Set(update.Message.From.ID, User{SteamID: steamID, AppID: -1})
					msg.Text = "Done"
					trySend(bot, msg)
				case "recent_games":
					u, ok := users.Get(update.Message.From.ID)
					if !ok {
						msg.Text = "You should register before using this command"
						trySend(bot, msg)
						return
					}

					r, err := getRecentGames(u.SteamID)
					if err != nil {
						log.Println(err)
						return
					}

					var buttons []tgbotapi.InlineKeyboardButton
					for _, g := range r.Response.Games {
						buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(g.Name, fmt.Sprintf("%v#?#?%v", g.AppID, g.Name)))
					}
					var keyboard = tgbotapi.NewInlineKeyboardMarkup(buttons)
					msg.Text = "Choose a game\n"
					msg.ReplyMarkup = keyboard
					trySend(bot, msg)

				case "achievements":
					u, ok := users.Get(update.Message.From.ID)
					if !ok || u.AppID == -1 {
						msg.Text = "You should register and choose a game before using this command"
						trySend(bot, msg)
						return
					}
					r, err := getAchievements(u.SteamID, u.AppID)
					if err != nil {
						log.Println(err)
						return
					}
					msg.Text = achievementsToText(r)
					msg.ParseMode = "HTML"

					trySend(bot, msg)

				case "status":
					var r common.HealthCheckResponse
					err := common.FetchJson("http://steam_backend:8000/status", &r)
					if err != nil {
						log.Println(err)
						msg.Text = err.Error()
					} else {
						b, _ := json.Marshal(r)
						msg.Text = fmt.Sprintf("```%v```", b)
					}
					trySend(bot, msg)
				}
			} else if update.CallbackQuery != nil {
				callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
				presser_id := update.CallbackQuery.From.ID
				requester_id := update.CallbackQuery.Message.ReplyToMessage.From.ID
				if presser_id != requester_id {
					callback.Text = "These are not your recent games!"
					if _, err := bot.Request(callback); err != nil {
						log.Println(err)
					}
					return
				}

				splitted := strings.SplitN(update.CallbackQuery.Data, "#?#?", 2)

				app_id, err := strconv.Atoi(splitted[0])
				if err != nil {
					log.Println(err)
					return
				}
				game_name := splitted[1]

				u, ok := users.Get(presser_id)
				if !ok {
					log.Printf("Failed to find the user %v", update.CallbackQuery.From.UserName)
					return
				}
				u.AppID = app_id
				users.Set(presser_id, u)

				callback.Text = "Game set!"
				if _, err := bot.Request(callback); err != nil {
					log.Println(err)
					return
				}
				m := update.CallbackQuery.Message

				edit := tgbotapi.NewEditMessageText(m.Chat.ID, m.MessageID, fmt.Sprintf("Chosen game: <b><a href=\"https://store.steampowered.com/app/%v\">%v</a></b>", app_id, game_name))
				edit.ParseMode = "HTML"

				trySend(bot, edit)
			}

		}(update)
	}
}
