package handlers

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	logs "log"
	"net/http"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/unswpcsoc/pcsocgo/commands"
)

type report struct {
	nilCommand
	Platform string `arg:"platform"`
}

func newReport() *report { return &report{} }

func (r *report) Aliases() []string { return []string{"report"} }

func (r *report) Desc() string { return "report root command." }

func (r *report) Subcommands() []commands.Command {
	return []commands.Command{newReportReply()}
}

func (r *report) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	return commands.NewSimpleSend(msg.ChannelID, commands.GetUsage(r)), nil
}

type reportReply struct {
	nilCommand
	Hash  string   `arg:"hash"`
	Reply []string `arg:"reply"`
}

func newReportReply() *reportReply { return &reportReply{} }

func (r *reportReply) Aliases() []string { return []string{"report reply"} }

func (r *reportReply) Desc() string { return "report reply" }

func (r *reportReply) Subcommands() []commands.Command {
	return []commands.Command{}
}

func (r *reportReply) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	// iterate all private channels in the state object
	cid := ""
	for _, cha := range ses.State.Ready.PrivateChannels {
		h := sha256.New()
		h.Write([]byte(cha.ID))
		hashString := fmt.Sprintf("%x", h.Sum(nil))
		if hashString == r.Hash {
			cid = cha.ID
		}
	}

	if cid == "" {
		return nil, errors.New("could not find channel with that hash")
	}

	// send the message
	reply := strings.Join(r.Reply, " ")
	_, err := ses.ChannelMessageSend(cid, reply)
	if err != nil {
		return nil, err
	}

	return commands.NewSimpleSend(msg.ChannelID, "Sent message: "+reply), nil
}

// ReportHandler handles the direct messages that come through from reports
func ReportHandler(ses *discordgo.Session, msg *discordgo.Message) {
	// generate hash
	h := sha256.New()
	h.Write([]byte(msg.ChannelID))
	hashString := fmt.Sprintf("%x", h.Sum(nil))

	// construct out message
	out := &discordgo.MessageSend{
		Content: "",
		TTS:     false,
		Files:   []*discordgo.File{},
	}

	out.Embed = &discordgo.MessageEmbed{
		Type:  discordgo.EmbedTypeRich,
		Title: "Report Received",
		Author: &discordgo.MessageEmbedAuthor{
			Name: hashString,
		},
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:   "Content:",
				Value:  msg.Content,
				Inline: false,
			},
		},
	}

	// handle attachments
	for _, attachment := range msg.Attachments {
		url := attachment.URL
		splits := strings.Split(url, ".")
		format := splits[len(splits)-1]
		logs.Println("report: Got attachment format: " + format)

		resp, err := http.Get(url)
		if err != nil {
			logs.Println("report:", err)
			continue
		}

		// read into buffer
		var buf = bytes.NewBuffer([]byte{})
		_, err = buf.ReadFrom(resp.Body)
		if err != nil {
			logs.Println("report:", err)
			continue
		}

		// close the response body
		resp.Body.Close()

		// chuck buffer into files
		out.Files = append(out.Files, &discordgo.File{
			Name:        "deleted attachment." + format,
			ContentType: "image/" + format,
			Reader:      buf,
		})
	}

	ses.ChannelMessageSendComplex(commands.Report.ID, out)
}
