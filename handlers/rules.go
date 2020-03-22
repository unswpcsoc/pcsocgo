package handlers

import (
	"github.com/bwmarrin/discordgo"

	"github.com/unswpcsoc/pcsocgo/commands"
	//"github.com/unswpcsoc/pcsocgo/internal/utils"
)

const (
	// PCSoc
	//rulesChannel = "602899198198808606"

	// PCSoc2
	rulesChannel = "602899198198808606"
)

type rules struct {
	nilCommand
}

func newRules() *rules { return &rules{} }

func (r *rules) Aliases() []string { return []string{"rules"} }

func (r *rules) Desc() string {
	return `[Verse: Jerma]
Rats, rats, we're the rats
We prey at night, we stalk at night, we're the rats
[King Rat]
I'm the giant rat that makes all of the rules
[All Rats]
Let's see what kind of trouble we can get ourselves into`
}

func (r *rules) Roles() []string { return []string{"mod"} }

func (r *rules) Chans() []string { return []string{"mods"} }

func (r *rules) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	return nil, nil
}

type rulesGet struct {
	nilCommand
	Index int `args:"index"`
}

func newRulesGet() *rulesGet { return &rulesGet{} }

func (r *rulesGet) Aliases() []string { return []string{"rules get"} }

func (r *rulesGet) Desc() string { return "Gets the current rules for index specified" }

func (r *rulesGet) Roles() []string { return []string{"mod"} }

func (r *rulesGet) Chans() []string { return []string{"mods"} }

func (r *rulesGet) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	// niceman
	return nil, nil
}

type rulesSet struct {
	nilCommand
	Index int `args:"index"`
}

func newRulesSet() *rulesSet { return &rulesSet{} }

func (r *rulesSet) Aliases() []string { return []string{"rules set"} }

func (r *rulesSet) Desc() string { return "Sets the rules for the index specified" }

func (r *rulesSet) Roles() []string { return []string{"mod"} }

func (r *rulesSet) Chans() []string { return []string{"mods"} }

func (r *rulesSet) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	// niceman
	return nil, nil
}
