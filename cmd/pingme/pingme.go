package main

import (
	"log"

	"github.com/unswpcsoc/pcsocgo/commands"
)

const (
	tagsKey = "fulltags"
)

// use tag implementation
type tag struct {
	UID      string
	Username string // don't trust this, always fetch from the UID
	Tag      string
	Platform string
	PingMe   bool
}

type platform struct {
	Name  string
	Role  interface{}
	Users map[string]*tag // indexed by user id's
}

// TODO: default games and api integrations
type tagStorer struct {
	Platforms map[string]*platform
}

func (t *tagStorer) Index() string { return "tags" }

func main() {
	var err error
	var tgs tagStorer

	err = commands.DBOpen("bot.db")
	if err != nil {
		log.Fatalln(err)
	}
	defer commands.DBClose()

	// open tags
	err = commands.DBGet(&tgs, tagsKey, &tgs)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Opened tags")

	// iterate tags
	for _, plat := range tgs.Platforms {
		log.Println("On Platform", plat)
		for _, tag := range plat.Users {
			tag.PingMe = true
		}
	}

	// confirm change
	for _, plat := range tgs.Platforms {
		for _, tag := range plat.Users {
			if !tag.PingMe {
				log.Fatalln("Change not successful")
			}
		}
	}

	// commit to the db
	commands.DBSet(&tgs, tagsKey)
	log.Println("Committed changes")
}
