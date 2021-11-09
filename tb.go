package main

import (
	"log"
	"strconv"
	"time"

	"gopkg.in/tucnak/telebot.v2"
)

type TB struct {
	i *telebot.Bot
}

func CreateTB() TB {
	tb, err := telebot.NewBot(telebot.Settings{
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
		i: tb,
	}
}

func (tb *TB) Start() {
	tb.i.Handle("/start", func(m *telebot.Message) {
		_, err := db.InitUser(*m.Sender)
		if err != nil {
			log.Fatal(err)
		}
		tb.SendMarkdown(m.Sender, c.TBM.Start)
		log.Println(m.Sender.Recipient())
	})

	tb.i.Handle("/search", func(m *telebot.Message) {
		user, err := db.InitUser(*m.Sender)
		if err != nil {
			log.Fatal(err)
		}

		if err := db.Search(user); err != nil {
			if err != ErrDBSearchAlreadyStarted {
				log.Fatal(err)
			}

			tb.SendMarkdown(m.Sender, "Поиск уже начат.")
		} else {
			tb.SendMarkdown(m.Sender, "Начинаю искать...")
		}

		// b.SendMarkdown(m.Sender, c.TBM.Start)
	})

	// b.i.Handle("/end", func(m *telebot.Message) {
	// 	b.SendMarkdown(m.Sender, c.TBM.Start)
	// })

	// b.i.Handle("/shareprofile", func(m *telebot.Message) {
	// 	b.SendMarkdown(m.Sender, c.TBM.Start)
	// })

	// b.i.Handle("/help", func(m *telebot.Message) {
	// 	b.SendMarkdown(m.Sender, c.TBM.Start)
	// })

	tb.i.Start()
}

func (tb *TB) SearchingComplete(users []string, roomNum []byte) error {
	for _, userID := range users {
		id, err := strconv.Atoi(userID)
		if err != nil {
			return err
		}

		user := &telebot.User{ID: id}

		tb.Send(user, "Добро пожаловать в #room"+strconv.FormatUint(bytesToUint64(roomNum), 10))
	}

	return nil
}

func (tb *TB) Send(to telebot.Recipient, m string) {
	_, err := tb.i.Send(to, m)

	if err != nil {
		log.Println(err)
	}
}

func (b *TB) SendMarkdown(to telebot.Recipient, m string) {
	_, err := tb.i.Send(to, m, telebot.ModeMarkdown)

	if err != nil {
		log.Println(err)
	}
}

func (b *TB) SendHTML(to telebot.Recipient, m string) {
	_, err := tb.i.Send(to, m, telebot.ModeHTML)

	if err != nil {
		log.Println(err)
	}
}
