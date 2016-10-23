package main

import (
	"encoding/json"
	"flag"
	//"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

type msgSend struct {
	bot *tgbotapi.BotAPI
	msg tgbotapi.MessageConfig
}

var cfg config
var sendMsgCh = make(chan *msgSend, 20)

func main() {
	var cfgfile string

	flag.StringVar(&cfgfile, "c", "config.json", "config file")
	flag.Parse()

	buf, err := ioutil.ReadFile(cfgfile)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(buf, &cfg)
	if err != nil {
		log.Fatal(err)
	}

	_, err = os.Stat(cfg.FileDir)
	if err != nil {
		err := os.Mkdir(cfg.FileDir, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	// limit is between 0 and 100
	if cfg.PollLimit > 100 || cfg.PollLimit < 0 {
		cfg.PollLimit = 100
	}

	if cfg.PollTimeout < 0 {
		cfg.PollTimeout = 60
	}

	if cfg.Debug {
		log.Printf("%+v\n", cfg)
	}

	bot, err := tgbotapi.NewBotAPIWithClient(
		cfg.Token,
		&http.Client{
			Timeout: time.Second * time.Duration(cfg.PollTimeout+10),
		})

	if err != nil {
		log.Fatal(err)
	}

	bot.Debug = cfg.Debug

	log.Printf("account %s\n", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = cfg.PollTimeout
	u.Limit = cfg.PollLimit

	updates, _ := bot.GetUpdatesChan(u)

	go sendMsg()

	for update := range updates {
		if update.UpdateID >= u.Offset {
			u.Offset = update.UpdateID + 1
		}

		if update.Message == nil {
			continue
		}
		//log.Printf("%s %s\n", update.Message.From.UserName, update.Message.Text)

		//msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		//bot.Send(msg)
		go handleUpdate(bot, update)
	}

}
