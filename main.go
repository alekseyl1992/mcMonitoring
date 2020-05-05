package main

import (
	"bufio"
	"fmt"
	"github.com/adamperlin/rcon"
	"github.com/caarlos0/env/v6"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var greetings []string

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

func getGreeter(cfg config, userName string) string {
	// get list of all custom named entities sorted by distance
	resultStr, err := execCommand(
		cfg, fmt.Sprintf(
			"/execute at %v run execute as @e[sort=nearest] at %v run data get entity @s CustomName",
			userName, userName))
	if err != err {
		log.Print(err)
		return ""
	}

	re := regexp.MustCompile(`(.*?) has the following`)
	groups := re.FindStringSubmatch(resultStr)
	if groups == nil {
		return ""
	}

	return groups[1]
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
			greeter := getGreeter(cfg, userName)

			greeting := greetings[rand.Intn(len(greetings))]
			greeting = fmt.Sprintf(greeting, userName)

			var err error
			if greeter != "" {
				_, err = execCommand(cfg, fmt.Sprintf(
					"/execute as @e[nbt={CustomName: '{\"text\":\"%v\"}'},limit=1] run say %v",
					greeter, greeting))
			} else {
				_, err = execCommand(cfg, fmt.Sprintf("/say %v", greeting))
			}
			if err != nil {
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

	DataPath string `env:"DATA_PATH" envDefault:"./data"`

	SleepInterval time.Duration `env:"SLEEP_INTERVAL" envDefault:"5s"`
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Panic(err)
	}

	rand.Seed(time.Now().UnixNano())

	file, err := os.Open(fmt.Sprintf("%v/greetings.txt", cfg.DataPath))
	if err != nil {
		log.Fatal(err)
	}
	// noinspection GoUnhandledErrorResult
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		greeting := strings.Trim(scanner.Text(), "\r\n ")
		if len(greeting) != 0 {
			greetings = append(greetings, greeting)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
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
