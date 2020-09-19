package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/unswpcsoc/pcsocgo/commands"
)

const (
	historyLim  = 2000
	archiveChan = "543714336401784862" // #archive
	scrollEmoji = string(0x1f4dc)
)

var (
	history = []*qelem{}
)

type qelem struct {
	cID string
	mID string
}

func enqueue(cid, mid string) {
	history = append(history, &qelem{cid, mid})
	if len(history) > historyLim {
		history = history[1:len(history)]
	}
}

type archive struct {
	nilCommand
	Index int `arg:"index"`
}

func newArchive() *archive { return &archive{} }

func (a *archive) Aliases() []string { return []string{"archive"} }

func (a *archive) Desc() string {
	return "Generates an embed for archiving a message.\nThe ordering for indexes is based on the order of messages reacted."
}

func (a *archive) Roles() []string { return []string{"mod"} }

func (a *archive) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	var err error

	if len(history) == 0 {
		return nil, errors.New("no logged messages have been reacted with " + scrollEmoji)
	}

	// check index
	if a.Index >= len(history) || a.Index < 0 {
		return nil, errors.New("index not in range")
	}

	cid := history[len(history)-a.Index-1].cID
	mid := history[len(history)-a.Index-1].mID

	// get archive target
	var arc *discordgo.Message
	arc, err = ses.State.Message(cid, mid)
	if err != nil {
		arc, err = ses.ChannelMessage(cid, mid)
		if err != nil {
			return nil, err
		}
	}

	var cha *discordgo.Channel
	cha, err = ses.State.Channel(arc.ChannelID)
	if err != nil {
		ses.Channel(arc.ChannelID)
		if err != nil {
			return nil, err
		}
	}

	// craft message
	out := &discordgo.MessageSend{
		Content: "",
		TTS:     false,
		Files:   []*discordgo.File{},
	}

	// add images, if they exist
	for i, attachment := range arc.Attachments {
		// get response, enqueue all attachments and assume image format
		url := attachment.URL
		splits := strings.Split(url, ".")
		format := splits[len(splits)-1]

		// func for the purpose of ez deferral
		func() {
			resp, err := http.Get(url)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			// read into buffer
			buf, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return
			}

			// attach to our message
			out.Files = append(out.Files, &discordgo.File{
				Name:        fmt.Sprintf("archive%d.%s", i, format),
				ContentType: "image/" + format,
				Reader:      bytes.NewReader(buf),
			})
		}()
	}

	// generate archive embed
	out.Embed = &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			URL:     fmt.Sprintf("https://discordapp.com/channels/%s/%s/%s", arc.GuildID, cid, mid),
			IconURL: arc.Author.AvatarURL(""),
			Name:    arc.Author.String(),
		},

		Description: arc.Content,

		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Archived message from %s | %s", cha.Name, arc.Timestamp),
		},

		Color: ses.State.UserColor(arc.Author.ID, cid),
	}

	// send to archive channel
	ses.ChannelMessageSendComplex(archiveChan, out)

	return commands.NewSimpleSend(msg.ChannelID, "Archived message!"), nil
}

func initArchive(ses *discordgo.Session) {
	ses.AddHandler(func(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
		react := r.MessageReaction
		if react.Emoji.Name == scrollEmoji {
			enqueue(react.ChannelID, react.MessageID)
		}
	})
}
