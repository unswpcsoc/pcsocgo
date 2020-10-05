// This package contains an implementation of pcsocgo
// using the provided utilities.
//
// You can make your own if you like I guess.
package main

import (
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"

	"github.com/unswpcsoc/pcsocgo/commands"
	"github.com/unswpcsoc/pcsocgo/handlers"
	"github.com/unswpcsoc/pcsocgo/internal/utils"
)

var (
	prod bool // production mode i.e. db saves to file rather than memory
	sync bool // sync mode - will handle events syncronously if set, might break things if you do this

	lastCom = make(map[string]commands.Command) // map of uid->command for most recently used command

	dgo *discordgo.Session

	errs = log.New(os.Stderr, "Error: ", log.Ltime) // logger for errors
)

// flag parse init
func init() {
	flag.BoolVar(&prod, "prod", false, "Enables production mode")
	flag.BoolVar(&sync, "sync", false, "Enables synchronous event handling")
	flag.Parse()
}

func main() {
	// logger init
	fp, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Println("Error opening logging file:", err)
	} else {
		defer fp.Close()
		log.SetOutput(io.MultiWriter(os.Stdout, fp))
	}

	// discordgo init
	key, ok := os.LookupEnv("KEY")
	if !ok {
		errs.Fatalln("Missing Discord API Key: Set env var $KEY")
	}

	dgo, err = discordgo.New("Bot " + key)
	if err != nil {
		errs.Fatalln(err)
	}

	err = dgo.Open()
	if err != nil {
		errs.Fatalln(err)
	}

	dgo.SyncEvents = sync

	log.Printf("Logged in as: %v\nSyncEvents is %v", dgo.State.User.ID, dgo.SyncEvents)
	defer dgo.Close()

	// db init
	if prod {
		err = commands.DBOpen("./bot.db")
	} else {
		err = commands.DBOpen(":memory:")
	}
	if err != nil {
		errs.Fatalln(err)
	}
	defer commands.DBClose()

	dgo.UpdateStatus(0, commands.Prefix+handlers.HelpAlias)

	// init loggers
	handlers.InitLogs(dgo)

	// init daemons
	var closeDaemons func()
	closeDaemons = handlers.InitDaemons(dgo)
	defer closeDaemons()

	// init guild cache
	err = commands.InitGuilds(dgo)
	if err != nil {
		errs.Fatalln(err)
	}
	log.Println("Operating on guild:", commands.Guild)

	// handle create message event
	dgo.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		handleMessageEvent(s, m.Message)
	})

	// handle update message event
	dgo.AddHandler(func(s *discordgo.Session, m *discordgo.MessageUpdate) {
		handleMessageEvent(s, m.Message)
	})

	// keep alive
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	sig := <-sc

	log.Println("Received Signal: " + sig.String())
	log.Println("Bye!")
}

func handleMessageEvent(s *discordgo.Session, m *discordgo.Message) {
	// catch panics on production
	if prod {
		defer func() {
			if r := recover(); r != nil {
				errs.Printf("Caught panic: %#v\n", r)
			}
		}()
	}

	if m.Author.ID == s.State.User.ID || m.Author.Bot {
		return
	}

	trm := strings.TrimSpace(m.Content)
	if !strings.HasPrefix(trm, commands.Prefix) || len(trm) == 1 {
		return
	}

	// route message
	var com commands.Command
	var ind int
	var ok bool
	argv := strings.Split(trm[1:], " ")
	if argv[0] == "!" {
		com, ok = lastCom[m.Author.ID]
		if !ok {
			return
		}
		// !! args...
		ind = 1
	} else {
		// regular routing
		com, ind = handlers.RouterRoute(argv)
		if com == nil {
			return
		}
	}

	// check chans
	chans := com.Chans()
	has, err := utils.MsgInChannels(s, m, chans)
	if err != nil {
		errs.Printf("Channel checking threw: %#v\n", err)
	}
	if !has {
		out := "Error: You must be in " + utils.Code(chans[0])
		if len(chans) > 1 {
			others := chans[1:]
			for _, oth := range others {
				out += " or " + utils.Code(oth)
			}
		}
		out += " to use this command"
		s.ChannelMessageSend(m.ChannelID, utils.Italics(out))
		return
	}

	// check roles
	roles := com.Roles()
	has, err = utils.MsgHasRoles(s, m, roles)
	if err != nil {
		errs.Printf("Role checking threw: %#v\n", err)
	}
	if !has {
		out := "Error: You must be a " + utils.Code(roles[0])
		if len(roles) > 1 {
			others := roles[1:]
			for _, oth := range others {
				out += " or a " + utils.Code(oth)
			}
		}
		out += " to use this command"
		s.ChannelMessageSend(m.ChannelID, utils.Italics(out))
		return
	}

	// successfully routed, register in !! before usage check
	lastCom[m.Author.ID] = com

	// fill args and check usage
	err = commands.FillArgs(com, argv[ind:])
	if err != nil {
		usage := "Usage: " + commands.GetUsage(com)
		s.ChannelMessageSend(m.ChannelID, usage)
		errs.Printf("Usage error on command %#v: %#v\n", com, err)
		return
	}

	// handle message
	log.Printf("Calling command handler: %+v", com)
	s.ChannelTyping(m.ChannelID)
	snd, err := com.MsgHandle(s, m)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, utils.Italics("Error: "+err.Error()))
		errs.Printf("%#v threw error: %#v\n", com, err)
		return
	}

	// send returned message
	if snd != nil {
		err = snd.Send(s)
		if err != nil {
			errs.Printf("Send error: %#v\n", err)
		}
	}

	// clean up args
	commands.CleanArgs(com)
}
