FROM golang:1.22-alpine

WORKDIR /steam_bot
COPY . .

WORKDIR /steam_bot/bot

RUN go mod download -x

RUN go build .

ENTRYPOINT ["./telegram_bot"]

