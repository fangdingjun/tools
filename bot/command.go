package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	//"log"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type cmdHandler func(bot *tgbotapi.BotAPI, update tgbotapi.Update, args string)

var defaultHandler = map[string]cmdHandler{}

func registerCommand(cmd string, fn cmdHandler) {
	defaultHandler[cmd] = fn
}

func getCommandHandler(cmd string) cmdHandler {
	if fn, ok := defaultHandler[cmd]; ok {
		return fn
	}
	return nil
}

func runHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update, args string) {
	// only trust user can run command
	if !isTrustUser(update.Message.From.UserName) {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "You don't authorized to run command")
		bot.Send(msg)
		return
	}

	c := exec.Command("sh", "-c", args)
	output, err := c.CombinedOutput()
	status := "exited with status 0"
	if err != nil {
		status = err.Error()
	}

	output = append(output, []byte("\n")...)
	output = append(output, []byte(status)...)

	l := len(output)
	for i := 0; i < (l / 4096); i++ {
		s1 := i * 4096
		s2 := (i + 1) * 4096
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, string(output[s1:s2]))
		bot.Send(msg)
	}

	r := l % 4096
	if r != 0 {
		r1 := l - r
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, string(output[r1:]))
		bot.Send(msg)
	}
}

func startHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update, args string) {
	s := fmt.Sprintf("Hi, %s", update.Message.From.FirstName)
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, s)
	bot.Send(msg)
}

func downloadHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update, args string) {
	// only trust user can download file
	if !isTrustUser(update.Message.From.UserName) {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "You don't authorized to download file")
		bot.Send(msg)
		return
	}

	if args == "" {
		m := tgbotapi.NewMessage(update.Message.Chat.ID, "usage:\n\t/download filepath")
		bot.Send(m)
		return
	}

	ss := strings.Fields(args)

	msg := ""
	for _, s := range ss {
		st, err := os.Stat(s)
		if err != nil {
			msg += fmt.Sprintf("%s not exists\n", s)
			continue
		}

		if !st.Mode().IsRegular() {
			msg += fmt.Sprintf("%s is not a regular file\n", s)
			continue
		}

		m := tgbotapi.NewDocumentUpload(update.Message.Chat.ID, s)
		bot.Send(m)
	}

	if msg != "" {
		m := tgbotapi.NewMessage(update.Message.Chat.ID, msg)
		bot.Send(m)
	}
}

func helpHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update, args string) {
	msg := `
	/run <cmd> <args>  run given command with arguments
	/download <file> download file
	/help  show this text
	/start show welcome message

	send the file or photo to the bot, the bot will download and store it
	`
	m := tgbotapi.NewMessage(update.Message.Chat.ID, msg)
	bot.Send(m)
}

func init() {
	registerCommand("run", runHandler)
	registerCommand("start", startHandler)
	registerCommand("download", downloadHandler)
	registerCommand("help", helpHandler)
}
