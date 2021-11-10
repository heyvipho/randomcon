package main

import (
	"log"
	"strconv"
	"time"

	"github.com/vipho/randomcon/database"
	"gopkg.in/tucnak/telebot.v2"
)

type TB struct {
	i  *telebot.Bot
	m  *Messages
	db DB
}

func CreateTB(token string, messages *Messages, db DB) (*TB, error) {
	i, err := telebot.NewBot(telebot.Settings{
		// You can also set custom API URL.
		// If field is empty it equals to "https://api.telegram.org".
		// URL: "http://195.129.111.17:8012",

		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})

	tb := TB{
		i:  i,
		m:  messages,
		db: db,
	}

	return &tb, err
}

func (tb *TB) Start() {
	tb.i.Handle("/start", func(m *telebot.Message) {
		_, err := tb.db.GetUser(m.Sender.ID)
		if err != nil {
			log.Fatal(err)
		}
		tb.SendMarkdown(m.Sender, tb.m.Start)
		log.Println(m.Sender.Recipient())
	})

	tb.i.Handle("/search", func(m *telebot.Message) {
		user, err := tb.db.GetUser(m.Sender.ID)
		if err != nil {
			log.Fatal(err)
		}

		findedUsers, err := tb.db.Search(user)
		if err != nil {
			if err != database.ErrDBSearchAlreadyStarted {
				log.Fatal(err)
			}

			tb.SendMarkdown(m.Sender, "Поиск уже начат.")

			return
		}

		tb.SendMarkdown(m.Sender, "Начинаю искать...")

		if len(findedUsers) > 0 {
			users := append(findedUsers, user.ID)

			roomNum, err := tb.db.AddRoom(users)
			if err != nil {
				log.Fatal(err)
			}

			for _, userID := range users {
				user := &telebot.User{ID: userID}
				tb.Send(user, "Добро пожаловать в #room"+strconv.FormatUint(roomNum, 10))
			}
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

func (tb *TB) Send(to telebot.Recipient, m string) {
	_, err := tb.i.Send(to, m)

	if err != nil {
		log.Println(err)
	}
}

func (tb *TB) SendMarkdown(to telebot.Recipient, m string) {
	_, err := tb.i.Send(to, m, telebot.ModeMarkdown)

	if err != nil {
		log.Println(err)
	}
}

func (tb *TB) SendHTML(to telebot.Recipient, m string) {
	_, err := tb.i.Send(to, m, telebot.ModeHTML)

	if err != nil {
		log.Println(err)
	}
}
