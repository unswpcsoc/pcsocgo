package handlers

import (
	logs "log"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/unswpcsoc/pcsocgo/commands"
)

const (
	bdaysKey = "birthday"
)

type birthdayStorer struct {
	birthdays map[string]time.Time
}

func (b *birthdayStorer) Index() string {
	return "birthday"
}

func initBirthday(ses *discordgo.Session) chan bool {
	logs.Println("Initialised birthday")

	ticker := time.NewTicker(time.Minute)
	done := make(chan bool)
	location, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		panic("location is bad :(")
	}

	now := time.Now()
	aestTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), location)

	doBirthday := func() {
		// check at aest midnight
		if !(aestTime.Hour() == 0 && aestTime.Minute() == 0) {
			return
		}
		// call handler
		logs.Println("Calling birthday handler")

		var bdays birthdayStorer
		err = commands.DBGet(&bdays, bdaysKey, &bdays)
		if err == commands.ErrDBNotFound {
			logs.Println("No birthdays found in db")
			return
		}

		// iterate birthdays
		for uid, bday := range bdays.birthdays {
			_, err := ses.User(uid)
			if err != nil {
				logs.Println("Could not find user with id:", uid)
			}
			logs.Println("bday for user is:", bday)
		}
	}

	go func() {
		// handle done signal
		select {
		case <-done:
			return
		default:
			doBirthday()
		}

		// handle ticker
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				doBirthday()
			}
		}
	}()
	return done
}
