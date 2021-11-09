package main

import (
	"log"
	"time"

	"gopkg.in/tucnak/telebot.v2"
)

type TB struct {
	i *telebot.Bot
}

func CreateTB() TB {
	b, err := telebot.NewBot(telebot.Settings{
		// You can also set custom API URL.
		// If field is empty it equals to "https://api.telegram.org".
		// URL: "http://195.129.111.17:8012",

		Token:  c.TBToken,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Panic(err)
	}

	return TB{
		i: b,
	}
}

func (b *TB) Start() {
	b.i.Handle("/start", func(m *telebot.Message) {
		_, err := db.InitUser(*m.Sender)
		if err != nil {
			log.Fatal(err)
		}
		b.SendMarkdown(m.Sender, c.TBM.Start)
		log.Println(m.Sender.Recipient())
	})

	// b.i.Handle("/search", func(m *telebot.Message) {
	// 	u, err := db.InitUser(*m.Sender)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	b.SendMarkdown(m.Sender, c.TBM.Start)
	// })

	// b.i.Handle("/end", func(m *telebot.Message) {
	// 	b.SendMarkdown(m.Sender, c.TBM.Start)
	// })

	// b.i.Handle("/shareprofile", func(m *telebot.Message) {
	// 	b.SendMarkdown(m.Sender, c.TBM.Start)
	// })

	// b.i.Handle("/help", func(m *telebot.Message) {
	// 	b.SendMarkdown(m.Sender, c.TBM.Start)
	// })

	b.i.Start()
}

func (b *TB) Send(to *telebot.User, m string) {
	_, err := b.i.Send(to, m)

	if err != nil {
		log.Println(err)
	}
}

func (b *TB) SendMarkdown(to *telebot.User, m string) {
	_, err := b.i.Send(to, m, telebot.ModeMarkdown)

	if err != nil {
		log.Println(err)
	}
}

func (b *TB) SendHTML(to *telebot.User, m string) {
	_, err := b.i.Send(to, m, telebot.ModeHTML)

	if err != nil {
		log.Println(err)
	}
}
