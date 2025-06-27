package main

import (
	"log"
	"os"
)

func main() {
	discordToken := os.Getenv("DISCORD_TOKEN")
	dbHost := os.Getenv("DB_HOST")
	log.Printf("Starting agis-bot with DISCORD_TOKEN=%s and DB_HOST=%s", discordToken, dbHost)
	// TODO: Add bot logic here
	select {} // keep running
}
