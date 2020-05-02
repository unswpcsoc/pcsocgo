// Package handlers contains concrete implementations of the Command interface
//
package handlers

import (
	"github.com/bwmarrin/discordgo"

	"github.com/unswpcsoc/pcsocgo/commands"
	"github.com/unswpcsoc/pcsocgo/internal/router"
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
