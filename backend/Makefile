DISCORD_TOKEN=

server:
	go build -o bin/server cmd/server/main.go && ./bin/server

discord:
	go build -o bin/discord cmd/discord/main.go && ./bin/discord -t $(DISCORD_TOKEN)

format:
	go build -o bin/format cmd/format/main.go && ./bin/format

.PHONY: server discord format
