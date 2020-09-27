// Package handlers contains concrete implementations of the Command interface
//
package handlers

import (
	"fmt"
	"math"

	"github.com/bwmarrin/discordgo"

	"github.com/unswpcsoc/pcsocgo/commands"
	"github.com/unswpcsoc/pcsocgo/internal/router"
)

const (
	emojiLeft          = "jrleft:681465381298503802"
	emojiRight         = "jrright:681465381356961827"
	fallbackEmojiLeft  = "⬅️"
	fallbackEmojiRight = "➡️"
)

var commandRouter *router.Router

func init() {
	commandRouter = router.NewRouter()

	commandRouter.AddCommand(newDecimalSpiral())

	commandRouter.AddCommand(newEcho())

	commandRouter.AddCommand(newHelp())

	commandRouter.AddCommand(newLog())
	commandRouter.AddCommand(newLogDelete())
	commandRouter.AddCommand(newLogFilter())

	commandRouter.AddCommand(newPing())

	commandRouter.AddCommand(newQuote())
	commandRouter.AddCommand(newQuoteAdd())
	commandRouter.AddCommand(newQuoteApprove())
	commandRouter.AddCommand(newQuoteList())
	commandRouter.AddCommand(newQuotePending())
	commandRouter.AddCommand(newQuoteRemove())
	commandRouter.AddCommand(newQuoteReject())
	commandRouter.AddCommand(newQuoteSearch())
	commandRouter.AddCommand(newQuoteClean())

	commandRouter.AddCommand(newRole("Bookworm"))
	commandRouter.AddCommand(newRole("Meta"))
	commandRouter.AddCommand(newRole("Weeb"))

	commandRouter.AddCommand(newTags())
	commandRouter.AddCommand(newTagsAdd())
	commandRouter.AddCommand(newTagsClean())
	commandRouter.AddCommand(newTagsGet())
	commandRouter.AddCommand(newTagsList())
	commandRouter.AddCommand(newTagsModRemove())
	commandRouter.AddCommand(newTagsPing())
	commandRouter.AddCommand(newTagsPingMe())
	commandRouter.AddCommand(newTagsPlatforms())
	commandRouter.AddCommand(newTagsRemove())
	commandRouter.AddCommand(newTagsShutup())
	commandRouter.AddCommand(newTagsUser())

	commandRouter.AddCommand(newArchive())

	commandRouter.AddCommand(newStaticIce())

	commandRouter.AddCommand(newHandbook())

	commandRouter.AddCommand(newScream())

	//commandRouter.AddCommand(newRules())
	//commandRouter.AddCommand(newRulesGet())
	//commandRouter.AddCommand(newRulesSet())

	commandRouter.AddCommand(newEmoji())
	commandRouter.AddCommand(newEmojiCount())
	commandRouter.AddCommand(newEmojiChungus())
	commandRouter.AddCommand(newEmojiCunt())
	commandRouter.AddCommand(newEmojiRegional())
}

// RouterRoute is a wrapper around the handler package's internal router's Route method
func RouterRoute(argv []string) (commands.Command, int) { return commandRouter.Route(argv) }

// RouterToSlice is a wrapper around the blah blah blah's ToSlice method
func RouterToSlice() []commands.Command { return commandRouter.ToSlice() }

// RouterToStringSlice is a wrapper around the blah blah blah's ToStringSlice method
func RouterToStringSlice() []string { return commandRouter.ToStringSlice() }

// nilCommand is a thing that you can struct embed to avoid boilerplate
type nilCommand struct{}

func (n *nilCommand) Subcommands() []commands.Command { return nil }

func (n *nilCommand) Roles() []string { return nil }

func (n *nilCommand) Chans() []string { return nil }

// InitLogs inits all logging commands.
// Needs to be maually updated when adding new loggers
func InitLogs(ses *discordgo.Session) {
	initFil(ses)
	initDel(ses)
	initArchive(ses)
	initEmoji(ses)
}

// InitDaemons inits all daemons, returns a function to close all channels when done
func InitDaemons(ses *discordgo.Session) (Close func()) {
	chans := []chan bool{}
	chans = append(chans, initClean(ses))
	return func() {
		// signal all channels on close
		for _, ch := range chans {
			ch <- true
		}
	}
}

// InitPaginated inits a reaction handler for a message to allow pagination
func InitPaginated(ses *discordgo.Session, msg *discordgo.Message, title string, lines []string, lineLimit int) (unregister func(), needUnregister bool) {
	// init return values
	unregister = nil
	needUnregister = false

	// keep state of message
	page := 0
	// pages are indexed at 0
	// Ceil(15/15)-1 = 0		| 15 15
	// Ceil(30/15)-1 = 1		| 15 15
	// Ceil(31/15)-1 = 2 		| 15 15 1
	lastPage := int(math.Ceil(float64(len(lines))/float64(lineLimit))) - 1
	lineLim := lineLimit
	once := false
	if len(lines) < lineLim {
		lastPage = 0
		lineLim = len(lines)
		once = true
	}

	// send a message first
	out := title
	for _, line := range lines[0:lineLim] {
		if line != "" {
			out += "\n" + line
		}
	}
	out += fmt.Sprintf("\n`Page 0/%d`", lastPage)

	// send initial message
	outMessage, err := ses.ChannelMessageSend(msg.ChannelID, out)
	if err != nil {
		return
	}

	// check once
	if once {
		return
	}

	// react with left and right emojis
	err = ses.MessageReactionAdd(msg.ChannelID, outMessage.ID, emojiLeft)
	if err != nil {
		// use fallback emoji
		err = ses.MessageReactionAdd(msg.ChannelID, outMessage.ID, fallbackEmojiLeft)
		if err != nil {
			return
		}
	}

	err = ses.MessageReactionAdd(msg.ChannelID, outMessage.ID, emojiRight)
	if err != nil {
		// use fallback emoji
		err = ses.MessageReactionAdd(msg.ChannelID, outMessage.ID, fallbackEmojiRight)
		if err != nil {
			return
		}
	}

	rootUnregister := ses.AddHandler(func(innerSes *discordgo.Session, event *discordgo.MessageReactionAdd) {
		reaction := event.MessageReaction

		// listen for reactions on the specific message sent
		if reaction.MessageID != outMessage.ID || reaction.UserID == outMessage.Author.ID {
			return
		}

		// ignore non-control emoji
		reactEmoji := reaction.Emoji.APIName()
		if reactEmoji != emojiLeft && reactEmoji != emojiRight && reactEmoji != fallbackEmojiLeft && reactEmoji != fallbackEmojiRight {
			return
		}

		// remove the reaction made by the user
		err := innerSes.MessageReactionRemove(
			reaction.ChannelID,
			reaction.MessageID,
			reaction.Emoji.APIName(),
			reaction.UserID,
		)
		if err != nil {
			fmt.Println(err)
			return
		}

		if reactEmoji == emojiLeft || reactEmoji == fallbackEmojiLeft {
			if page == 0 {
				page = lastPage
			} else {
				page--
			}
		}

		if reactEmoji == emojiRight || reactEmoji == fallbackEmojiRight {
			if page+1 > lastPage {
				page = 0
			} else {
				page++
			}
		}

		// calculate bounds
		left := page * lineLimit
		right := (page + 1) * lineLimit
		if right > len(lines) {
			right = len(lines)
		}

		// construct edit message
		edit := title
		for _, line := range lines[left:right] {
			if line != "" {
				edit += "\n" + line
			}
		}
		edit += fmt.Sprintf("\n`Page %d/%d`", page, lastPage)

		// actually edit the damn message
		innerSes.ChannelMessageEdit(reaction.ChannelID, reaction.MessageID, edit)
	})

	unregister = func() {
		ses.MessageReactionsRemoveAll(msg.ChannelID, outMessage.ID)
		ses.MessageReactionsRemoveAll(msg.ChannelID, outMessage.ID)
		rootUnregister()
	}

	// set needs unregister
	needUnregister = true
	return
}
