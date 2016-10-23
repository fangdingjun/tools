package main

import (
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func handleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	msg := update.Message

	cmd := msg.Command()
	if cmd != "" {
		args := msg.CommandArguments()
		handleCommand(bot, update, cmd, args)
		return
	}

	handleMsg(bot, msg)
}

func handleCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update, cmd, args string) {

	handler := getCommandHandler(cmd)
	if handler != nil {
		handler(bot, update, args)
		return
	}
	s := fmt.Sprintf("Hello, %s", update.Message.From.FirstName)
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, s)

	sendMsgCh <- &msgSend{bot, msg}
}

func handleMsg(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	fileID := ""
	fileName := ""

	switch {
	case msg.Text != "":
		m := tgbotapi.NewMessage(msg.Chat.ID, msg.Text)
		//bot.Send(m)
		sendMsgCh <- &msgSend{bot, m}
	case msg.Document != nil:
		fileID = msg.Document.FileID
		fileName = msg.Document.FileName
	case msg.Video != nil:
		fileID = msg.Video.FileID
	case msg.Photo != nil:
		size := 0
		for _, p := range *msg.Photo {
			if p.Width > size {
				fileID = p.FileID
				size = p.Width
			}
		}
	case msg.Voice != nil:
		fileID = msg.Voice.FileID
	case msg.Audio != nil:
		fileID = msg.Audio.FileID
	default:
		s := "Welcome"
		m := tgbotapi.NewMessage(msg.Chat.ID, s)
		//bot.Send(m)
		sendMsgCh <- &msgSend{bot, m}
	}

	if fileID != "" {
		url, err := bot.GetFileDirectURL(fileID)
		if err != nil {
			log.Println(err)
			return
		}

		name := ""
		if fileName != "" {
			name = filepath.Join(cfg.FileDir, fileName)
			_, err := os.Stat(name)
			if err == nil { // file exists
				n := fmt.Sprintf("1_%d_%s", time.Now().Unix(), fileName)
				name = filepath.Join(cfg.FileDir, n)
			}
		} else {
			ext := filepath.Ext(url)
			n := fmt.Sprintf("1_%d%s", time.Now().Unix(), ext)
			name = filepath.Join(cfg.FileDir, n)
		}

		download(url, name)

		s := fmt.Sprintf("saved as %s", name)
		m := tgbotapi.NewMessage(msg.Chat.ID, s)
		//bot.Send(m)
		sendMsgCh <- &msgSend{bot, m}
	}

}

func isTrustUser(u string) bool {
	for _, u1 := range cfg.TrustUsers {
		if u1 == u {
			return true
		}
	}
	return false
}

func download(url, save string) {
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return
	}

	defer resp.Body.Close()

	fp, err := os.Create(save)
	if err != nil {
		log.Println(err)
		return
	}

	defer fp.Close()

	buf := make([]byte, 4096)

	for {
		n, err := resp.Body.Read(buf[0:])
		if n > 0 {
			fp.Write(buf[:n])
		}
		if err != nil {
			if err != io.EOF {
				log.Println(err)
			}
			break
		}
	}
}

func sendMsg() {
	for msg := range sendMsgCh {
		for i := 0; i < 10; i++ {
			_, err := msg.bot.Send(msg.msg)
			if err == nil {
				break
			}
			time.Sleep(3 * time.Second)
		}
	}
}
