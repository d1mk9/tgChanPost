package main

import (
	"github.com/d1mk9/tgChanPost/configs"
	"github.com/d1mk9/tgChanPost/internal/bot"
)

func main() {
	configs.LoadConfig()
	bot.StartBot()
}
