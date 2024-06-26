FROM golang:1.22-alpine

WORKDIR /steam_bot
COPY . .

WORKDIR /steam_bot/backend

RUN go mod download -x

RUN go build .

ENTRYPOINT ["./steam_api_service"]

