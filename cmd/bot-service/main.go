package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"linkedout/internal/bot"
	"linkedout/internal/config"
	"linkedout/internal/llm"
	"linkedout/internal/logger"
	"linkedout/internal/metricsclient"
)

func main() {
	cfg := config.Load()
	log := logger.New("bot-service")

	if cfg.BotToken == "" {
		log.Error("BOT_TOKEN is empty")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	httpClient := &http.Client{Timeout: cfg.HTTPClientTimeout}
	telegramBot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Error("create telegram bot failed", "error", err)
		os.Exit(1)
	}

	metricsClient := metricsclient.New(cfg.MetricsBaseURL, httpClient, log)
	llmClient := llm.NewClient(cfg.LLMAPIKey, cfg.LLMBaseURL, cfg.LLMModel, httpClient, log)
	service := bot.NewService(telegramBot, llmClient, metricsClient, cfg.BotAdminIDs, log)

	log.Info("bot-service started", "bot_username", telegramBot.Self.UserName)
	if err := service.Run(ctx); err != nil {
		log.Error("bot-service stopped with error", "error", err)
		os.Exit(1)
	}
}
