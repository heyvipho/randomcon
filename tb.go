package main

import (
	"bytes"
	"log"
	"regexp"
	"strconv"
	"time"

	"text/template"

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
		log.Println(m.Sender.Recipient())
		_, err := tb.db.GetUser(m.Sender.ID)
		if err != nil {
			log.Fatal(err)
		}
		if err := tb.SendMarkdown(m.Sender, tb.m.Start); err != nil {
			tb.minorErr(m.Sender, err)
			return
		}
	})

	tb.i.Handle("/search", func(m *telebot.Message) {
		if err := tb.search(m); err != nil {
			log.Fatal(err)
		}
	})

	tb.i.Handle("/unsearch", func(m *telebot.Message) {
		if err := tb.unSearch(m); err != nil {
			log.Fatal(err)
		}
	})

	tb.i.Handle("/leave", func(m *telebot.Message) {
		if err := tb.leaveRoom(m); err != nil {
			log.Fatal(err)
		}
	})

	// b.i.Handle("/shareprofile", func(m *telebot.Message) {
	// 	b.SendMarkdown(m.Sender, c.TBM.Start)
	// })

	tb.i.Handle("/help", func(m *telebot.Message) {
		if err := tb.SendMarkdown(m.Sender, tb.m.Start); err != nil {
			tb.minorErr(m.Sender, err)
			return
		}
	})

	tb.i.Handle(telebot.OnText, func(m *telebot.Message) {
		if tb.IsCommand(m.Text) {
			return
		}
		tb.forward(m.Sender, m.Text)
	})

	tb.i.Handle(telebot.OnPhoto, func(m *telebot.Message) { tb.forward(m.Sender, m.Photo) })
	tb.i.Handle(telebot.OnAudio, func(m *telebot.Message) { tb.forward(m.Sender, m.Audio) })
	tb.i.Handle(telebot.OnAnimation, func(m *telebot.Message) { tb.forward(m.Sender, m.Animation) })
	tb.i.Handle(telebot.OnDocument, func(m *telebot.Message) { tb.forward(m.Sender, m.Document) })
	tb.i.Handle(telebot.OnSticker, func(m *telebot.Message) { tb.forward(m.Sender, m.Sticker) })
	tb.i.Handle(telebot.OnVideo, func(m *telebot.Message) { tb.forward(m.Sender, m.Video) })
	tb.i.Handle(telebot.OnVoice, func(m *telebot.Message) { tb.forward(m.Sender, m.Voice) })
	tb.i.Handle(telebot.OnVideoNote, func(m *telebot.Message) { tb.forward(m.Sender, m.VideoNote) })
	tb.i.Handle(telebot.OnLocation, func(m *telebot.Message) { tb.forward(m.Sender, m.Location) })
	tb.i.Handle(telebot.OnVenue, func(m *telebot.Message) { tb.forward(m.Sender, m.Venue) })
	tb.i.Handle(telebot.OnDice, func(m *telebot.Message) { tb.forward(m.Sender, m.Dice) })
	tb.i.Handle(telebot.OnInvoice, func(m *telebot.Message) { tb.forward(m.Sender, m.Invoice) })
	tb.i.Handle(telebot.OnPoll, func(m *telebot.Message) { tb.forward(m.Sender, m.Poll) })

	tb.i.Start()
}

func (tb *TB) forward(sender *telebot.User, what interface{}) {
	user, err := tb.db.GetUser(sender.ID)
	if err != nil {
		log.Fatal(err)
	}

	room, err := tb.db.GetRoom(user.CurrentRoom)
	if err != nil {
		if err != database.ErrKeyNotFound {
			log.Fatal(err)
		}

		if err := tb.SendMarkdown(sender, tb.m.RoomNotInside); err != nil {
			tb.minorErr(sender, err)
			return
		}

		return
	}

	for _, v := range room.Users {
		if v != user.ID {
			to := &telebot.User{ID: v}

			if _, err := tb.i.Send(to, what); err != nil {
				tb.minorErr(to, err)
				continue
			}
		}
	}
}

func (tb *TB) unSearch(m *telebot.Message) error {
	user, err := tb.db.GetUser(m.Sender.ID)
	if err != nil {
		return err
	}

	yes, err := tb.db.IsSearching(user)
	if err != nil {
		return err
	}
	if !yes {
		if err := tb.SendMarkdown(m.Sender, tb.m.SearchYouAreNotSearching); err != nil {
			tb.minorErr(m.Sender, err)
			return nil
		}

		return nil
	}

	if err := tb.db.UnSearch(user); err != nil {
		return err
	}

	if err := tb.SendMarkdown(m.Sender, tb.m.SearchStopped); err != nil {
		tb.minorErr(m.Sender, err)
		return nil
	}

	return nil
}

func (tb *TB) search(m *telebot.Message) error {
	user, err := tb.db.GetUser(m.Sender.ID)
	if err != nil {
		return err
	}

	yes, err := tb.db.IsRoomExist(user.CurrentRoom)
	if err != nil {
		return err
	}
	if yes {
		if err := tb.delRoom(user.CurrentRoom); err != nil {
			return err
		}
	}

	findedUsers, err := tb.db.Search(user)
	if err != nil {
		if err != database.ErrDBSearchAlreadyStarted {
			return err
		}

		if err := tb.SendMarkdown(m.Sender, tb.m.SearchAlreadyStarted); err != nil {
			return err
		}

		return nil
	}

	if err := tb.SendMarkdown(m.Sender, tb.m.SearchStarting); err != nil {
		tb.minorErr(m.Sender, err)
		return err
	}

	if len(findedUsers) > 0 {
		users := append(findedUsers, user.ID)

		roomNum, err := tb.db.AddRoom(users)
		if err != nil {
			return err
		}

		for _, userID := range users {
			user := &telebot.User{ID: userID}

			mv := map[string]string{
				"RoomNum": strconv.FormatUint(roomNum, 10),
			}
			message, err := tb.Template(tb.m.RoomWelcome, mv)
			if err != nil {
				tb.minorErr(user, err)
				continue
			}

			if err := tb.SendMarkdown(user, message); err != nil {
				tb.minorErr(user, err)
				continue
			}
		}
	}

	return nil
}

func (tb *TB) leaveRoom(m *telebot.Message) error {
	user, err := tb.db.GetUser(m.Sender.ID)
	if err != nil {
		log.Fatal(err)
	}

	yes, err := tb.db.IsRoomExist(user.CurrentRoom)
	if err != nil {
		return err
	}
	if !yes {
		if err := tb.SendMarkdown(m.Sender, tb.m.RoomNotInside); err != nil {
			tb.minorErr(m.Sender, err)
			return nil
		}
	}

	err = tb.delRoom(user.CurrentRoom)
	return err
}

func (tb *TB) delRoom(roomNum uint64) error {
	userIDs, err := tb.db.DelRoom(roomNum)
	if err != nil {
		return err
	}

	for _, userID := range userIDs {
		user := &telebot.User{ID: userID}

		mv := map[string]string{
			"RoomNum": strconv.FormatUint(roomNum, 10),
		}
		message, err := tb.Template(tb.m.RoomDestroy, mv)
		if err != nil {
			tb.minorErr(user, err)
			continue
		}

		if err := tb.SendMarkdown(user, message); err != nil {
			tb.minorErr(user, err)
			continue
		}
	}

	return nil
}

func (tb *TB) SendMarkdown(to telebot.Recipient, m string) error {
	_, err := tb.i.Send(to, m, telebot.ModeMarkdown)

	return err
}

func (tb *TB) SendHTML(to telebot.Recipient, m string) error {
	_, err := tb.i.Send(to, m, telebot.ModeHTML)

	return err
}

func (tb *TB) Template(text string, values map[string]string) (string, error) {
	t := template.Must(template.New("text").Parse(text))

	var s string
	buf := bytes.NewBufferString(s)

	err := t.Execute(buf, values)

	return buf.String(), err
}

func (tb *TB) minorErr(to telebot.Recipient, errors ...interface{}) {
	log.Println(errors...)

	if err := tb.SendMarkdown(to, tb.m.MinorError); err != nil {
		log.Println(err)
	}
}

func (tb *TB) IsCommand(s string) bool {
	cmdRx := regexp.MustCompile(`^(/\w+)(@(\w+))?(\s|$)(.+)?`)
	return cmdRx.MatchString(s)
}
