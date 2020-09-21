package handlers

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/unswpcsoc/pcsocgo/commands"
	"github.com/unswpcsoc/pcsocgo/internal/utils"
)

const (
	keyEmoji      = "emoji"
	thinkingEmoji = string(0x1f914)
)

var (
	ErrEmojiNotInit = errors.New("emoji counter not initialised")
)

// emojis implements the Storer interface
type emojis struct {
	Counter map[string]int
	Start   time.Time
}

func (e *emojis) Index() string {
	return keyEmoji
}

// emoji implements the Command interface
type emoji struct {
	nilCommand
}

func newEmoji() *emoji { return &emoji{} }

func (e *emoji) Subcommands() []commands.Command {
	return []commands.Command{
		newEmojiCount(),
		newEmojiChungus(),
		newEmojiCunt(),
		newEmojiRegional(),
	}
}

func (e *emoji) Aliases() []string { return []string{"emoji", thinkingEmoji} }

func (e *emoji) Desc() string { return "Prints a random custom server emoji" }

func (e *emoji) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	// get guild emojis
	emojis, err := ses.GuildEmojis(msg.GuildID)
	if err != nil {
		return nil, err
	}

	// randomly select an emoji
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	picked := emojis[r.Intn(len(emojis))].MessageFormat()

	return commands.NewSimpleSend(msg.ChannelID, picked), nil
}

type emojiCount struct {
	nilCommand
}

func newEmojiCount() *emojiCount { return &emojiCount{} }

func (e *emojiCount) Aliases() []string {
	return []string{"emoji count", "emoji co", "emoji stats", "emoji st"}
}

func (e *emojiCount) Desc() string { return "Prints a summary of the usage of custom server emojis" }

func (e *emojiCount) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	// Get emojis
	var emo emojis
	err := commands.DBGet(&emojis{}, keyEmoji, &emo)
	if err == commands.ErrDBNotFound {
		return nil, ErrEmojiNotInit
	} else if err != nil {
		return nil, err
	}

	var out string = "Emoji stats (from " + emo.Start.Format("15:04:05 MST 2006-01-02") + "):\n"
	for emojiText, count := range emo.Counter {
		out += fmt.Sprintf("%s : %d\n", emojiText, count)
	}

	return commands.NewSimpleSend(msg.ChannelID, out), nil
}

type emojiChungus struct {
	nilCommand
	Emoji []string `arg:"emoji"`
}

func newEmojiChungus() *emojiChungus { return &emojiChungus{} }

func (e *emojiChungus) Aliases() []string { return []string{"emoji chungus", "emoji ch", "chungus"} }

func (e *emojiChungus) Desc() string {
	return "Prints a chungus with the emoji supplied or a random custom server one"
}

func (e *emojiChungus) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	// get guild emojis
	emojis, err := ses.GuildEmojis(msg.GuildID)
	if err != nil {
		return nil, err
	}

	var picked string
	if len(e.Emoji) == 0 {
		// randomly select an emoji
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		picked = emojis[r.Intn(len(emojis))].MessageFormat()
	} else {
		picked = strings.Join(e.Emoji, " ")
	}

	chungachunga := "<:cw:590153701252005907><:c1_0:590153698324381698><:c2_0:590153703609204760><:cw:590153701252005907>\n<"
	chungachunga += ":cw:590153701252005907><:c1_1:590153699372826634>" + picked + "<:c3_1:590153701281497109>\n"
	chungachunga += "<:c0_2:590153690493747220><:c1_2:590153704129298442><:c2_2:590153702443319307><:c3_2:590153704121040898>\n"
	chungachunga += "<:c0_3:590153695363203072><:c1_3:590153703454015493><:c2_3:590153701969231873><:c3_3:590153703697416192>"

	return commands.NewSimpleSend(msg.ChannelID, chungachunga), nil
}

type emojiCunt struct {
	nilCommand
}

func newEmojiCunt() *emojiCunt { return &emojiCunt{} }

func (e *emojiCunt) Aliases() []string { return []string{"emoji cunt"} }

func (e *emojiCunt) Desc() string { return "OI" }

func (e *emojiCunt) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	return commands.NewSimpleSend(msg.ChannelID, utils.EmojiAlpha("OI CUNT")), nil
}

type emojiRegional struct {
	nilCommand
	Message []string `arg:"Message"`
}

func newEmojiRegional() *emojiRegional { return &emojiRegional{} }

func (e *emojiRegional) Aliases() []string { return []string{"emoji regional", "regional"} }

func (e *emojiRegional) Desc() string { return "Returns alphanumeric messages" }

func (e *emojiRegional) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	return commands.NewSimpleSend(msg.ChannelID, utils.EmojiAlpha(strings.Join(e.Message, " "))), nil
}

// logger for emoji count
func initEmoji(ses *discordgo.Session) {
	ses.AddHandler(func(se *discordgo.Session, mc *discordgo.MessageCreate) {

		// lock the db for writing
		commands.DBLock()
		defer commands.DBUnlock()

		if mc.Author.Bot {
			return
		}

		var emo emojis
		// get the emoji list
		err := commands.DBGet(&emojis{}, keyEmoji, &emo)
		if err == commands.ErrDBNotFound {
			// create a new emoji list
			emo = emojis{
				Counter: make(map[string]int),
				Start:   time.Now(),
			}
		} else if err != nil {
			return
		}

		// get guild emojis
		var emojis []*discordgo.Emoji
		emojis, err = ses.GuildEmojis(mc.GuildID)
		if err != nil {
			return
		}

		// check message for emojis
		for _, emoji := range emojis {
			var emojiText string = emoji.MessageFormat()
			if strings.Contains(mc.Content, emojiText) {
				emo.Counter[emojiText] += 1
			}
		}

		// Set the in the db
		commands.DBSet(&emo, keyEmoji)
	})

	ses.AddHandler(func(se *discordgo.Session, mra *discordgo.MessageReactionAdd) {

		// lock the db for writing
		commands.DBLock()
		defer commands.DBUnlock()

		mem, err := se.State.Member(mra.GuildID, mra.UserID)
		if err == nil {
			if mem.User.Bot {
				return
			}
		} else {
			// fall back to session
			usr, err := se.User(mra.UserID)
			if err != nil {
				return
			}
			if usr.Bot {
				return
			}
		}

		var emo emojis
		// get the emoji list
		err = commands.DBGet(&emojis{}, keyEmoji, &emo)
		if err == commands.ErrDBNotFound {
			// create a new emoji list
			emo = emojis{
				Counter: make(map[string]int),
				Start:   time.Now(),
			}
		} else if err != nil {
			return
		}

		// get guild emojis
		var emojis []*discordgo.Emoji
		emojis, err = ses.GuildEmojis(mra.GuildID)
		if err != nil {
			return
		}

		// check reaction
		for _, emoji := range emojis {
			var emojiText string = emoji.MessageFormat()
			if mra.Emoji.MessageFormat() == emoji.MessageFormat() {
				emo.Counter[emojiText] += 1
			}
		}

		// Set the in the db
		commands.DBSet(&emo, keyEmoji)
	})

	ses.AddHandler(func(se *discordgo.Session, mrr *discordgo.MessageReactionRemove) {

		// lock the db for writing
		commands.DBLock()
		defer commands.DBUnlock()

		mem, err := se.State.Member(mrr.GuildID, mrr.UserID)
		if err == nil {
			if mem.User.Bot {
				return
			}
		} else {
			// fall back to session
			usr, err := se.User(mrr.UserID)
			if err != nil {
				return
			}
			if usr.Bot {
				return
			}
		}

		var emo emojis
		// get the emoji list
		err = commands.DBGet(&emojis{}, keyEmoji, &emo)
		if err == commands.ErrDBNotFound {
			return
		} else if err != nil {
			return
		}

		// get guild emojis
		var emojis []*discordgo.Emoji
		emojis, err = ses.GuildEmojis(mrr.GuildID)
		if err != nil {
			return
		}

		// check reaction
		for _, emoji := range emojis {
			var emojiText string = emoji.MessageFormat()
			if mrr.Emoji.MessageFormat() == emoji.MessageFormat() {
				if emo.Counter[emojiText] > 0 {
					emo.Counter[emojiText] -= 1
				}
			}
		}

		// Set the in the db
		commands.DBSet(&emo, keyEmoji)
	})
	return
}
