module steam_api_service

go 1.22.3

replace common => ../common

require (
	common v0.0.0-00010101000000-000000000000
	github.com/gorilla/mux v1.8.1
	golang.org/x/sync v0.7.0
)
