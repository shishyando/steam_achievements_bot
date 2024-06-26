# Steam achievements bot

This is a simple telegram bot + steam api duo made out of boredom when I wanted to collect all Terraria achievements (I did it, actually).

### Usage

Just put your steam API key in `./backend/api_key` and your telegram bot's token in `./bot/telegram_bot_token`, then run `docker compose`.

### Steam API

A separate microservice that exports an API to get user's steam id by login, recent games and user's achievements info for `app_id`.

NB: it should fetch all achievements with their images' urls, but it's not used in any way. Maybe you could download the images and store them nicely in some local DB and then show on your fancy website or whatever. I didn't bother doing that in telegram.

Steam API key required, get it yourself.

### Telegram bot

A telegram bot that uses the service above to get all needed info, wraps in some simple UI like buttons and stuff. Some basic data storage for convenience included.

Token required, get it yourself.

