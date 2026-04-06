package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nczz/cron-sidecar/bot"
	"github.com/nczz/cron-sidecar/locale"
)

func main() {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_BOT_TOKEN is required")
	}
	kiroCLI := os.Getenv("KIRO_CLI_PATH")
	if kiroCLI == "" {
		kiroCLI = "kiro-cli"
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/home/agent/.kiro/cron-data"
	}
	guildID := os.Getenv("DISCORD_GUILD_ID")
	tz := os.Getenv("CRON_TIMEZONE")
	if tz == "" {
		tz = "Asia/Taipei"
	}
	botLocale := os.Getenv("BOT_LOCALE")
	if botLocale == "" {
		botLocale = "zh-TW"
	}

	locale.Load(botLocale)

	b, err := bot.New(bot.Config{
		Token:    token,
		KiroCLI:  kiroCLI,
		DataDir:  dataDir,
		GuildID:  guildID,
		Timezone: tz,
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := b.Start(ctx); err != nil {
		log.Fatal(err)
	}
	log.Println("cron-sidecar running")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc

	cancel()
	b.Stop()
}
