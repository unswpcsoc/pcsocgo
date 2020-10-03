package handlers

import (
	"bwmarrin/discordgo"
	"logs"
	"time"
)

func initBirthday(ses *discordgo.Session) chan bool {
	logs.Println("Initialised birthday")

	ticker := time.NewTicker(time.Minute)
	done := make(chan bool)
	location, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		return false
	}
	now := time.Now()
	aestTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), location)

	doBirthday := func() {
		// check at aest midnight
		if aestTime.Hour() == 0 && aestTime.Minute() == 0 {
			// call handler
			logs.Println("Calling birthday handler")
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
