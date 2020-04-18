package main

import (
	"fmt"
	"github.com/adamperlin/rcon"
	"github.com/caarlos0/env/v6"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func execCommand(cfg config, command string) (string, error) {
	client, err := rcon.NewClient(cfg.RconHost, cfg.RconPort)

	if nil != err {
		log.Print("Failed to open TCP connection to server.")
		return "", err
	}

	var packet *rcon.Packet

	packet, err = client.Authorize(cfg.RconPass)

	if nil != err {
		log.Print("Failed to authorize your connection with the server.")
		return "", err
	}

	packet, err = client.Execute(command)

	if nil != err {
		log.Print("Failed to execute command")
		return "", err
	}

	return packet.Body, nil
}

func difference(a, b []string) (diff []string) {
	m := make(map[string]bool)

	for _, item := range b {
		m[item] = true
	}

	for _, item := range a {
		if _, ok := m[item]; !ok {
			diff = append(diff, item)
		}
	}
	return
}

func newMessageWithParseMode(chatID int64, text string, parseMode string) tgbotapi.MessageConfig {
	message := tgbotapi.NewMessage(chatID, text)
	message.ParseMode = parseMode

	return message
}

func iteration(cfg config, bot *tgbotapi.BotAPI, lastPlayersPtr *[]string, firstRun *bool) {
	lastPlayers := *lastPlayersPtr

	resultStr, err := execCommand(cfg, "/list")
	if err != err {
		log.Print(err)
		return
	}

	splitRes := strings.Split(resultStr, ": ")

	var players []string
	if splitRes[1] != "" {
		players = strings.Split(splitRes[1], ", ")
	} else {
		players = make([]string, 0)
	}

	log.Printf("Players: %v", players)

	if !*firstRun {
		joinedPlayers := difference(players, lastPlayers)

		for _, userName := range joinedPlayers {
			_, err := execCommand(cfg, fmt.Sprintf("/say Hi, %v! Good to see you again.", userName))
			if err != err {
				log.Print(err)
				return
			}

			_, err = bot.Send(newMessageWithParseMode(
				cfg.TgChatId,
				fmt.Sprintf("*%v* joined the server", userName),
				"markdown"))

			if err != nil {
				log.Print(err)
				return
			}
		}

		leftPlayers := difference(lastPlayers, players)
		for _, userName := range leftPlayers {
			_, err = bot.Send(newMessageWithParseMode(
				cfg.TgChatId,
				fmt.Sprintf("*%v* left the server", userName),
				"markdown"))

			if err != nil {
				log.Print(err)
				return
			}
		}
	}

	*lastPlayersPtr = players
	*firstRun = false
}

type config struct {
	TgToken  string `env:"TG_TOKEN,required"`
	TgChatId int64  `env:"TG_CHAT_ID,required"`
	TgProxy  string `env:"TG_PROXY"`

	RconHost string `env:"RCON_HOST"`
	RconPort int    `env:"RCON_PORT"`
	RconPass string `env:"RCON_PASS"`

	SleepInterval time.Duration `env:"SLEEP_INTERVAL" envDefault:"5s"`
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Panic(err)
	}

	var client http.Client
	if cfg.TgProxy != "" {
		proxyUrl, err := url.Parse(cfg.TgProxy)
		if err != nil {
			panic(err)
		}

		client = http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}
	} else {
		client = http.Client{Transport: &http.Transport{}}
	}

	bot, err := tgbotapi.NewBotAPIWithClient(cfg.TgToken, &client)
	if err != nil {
		panic(err)
	}

	log.Print("Started")

	lastPlayers := make([]string, 0)
	firstRun := true

	for {
		iteration(cfg, bot, &lastPlayers, &firstRun)
		time.Sleep(cfg.SleepInterval)
	}
}
