module telegram_bot

go 1.22.3

replace common => ../common

require (
	common v0.0.0-00010101000000-000000000000
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/orcaman/concurrent-map/v2 v2.0.1
)
