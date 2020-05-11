package handlers

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/unswpcsoc/pcsocgo/commands"
	"github.com/unswpcsoc/pcsocgo/internal/utils"
)

const (
	keyPending = "pending"
	keyQuotes  = "approve"

	quoteListLineLimit = 80
	quoteListLimit     = 15

	searchLimit = 5

	// ��
	emojiLeft  = "jrleft:681465381298503802"
	emojiRight = "jrright:681465381356961827"
	//emojiLeft  = "leee:690176095433392149"
	//emojiRight = "reee:468260188500131850"
)

var (
	// ErrQuoteIndex means quote index is not valid
	ErrQuoteIndex = errors.New("index not valid")
	// ErrQuoteEmpty means quote list is not there
	ErrQuoteEmpty = errors.New("quote list not initialised")
	// ErrQuoteNone means user entered no quote
	ErrQuoteNone = errors.New("no quote entered, please enter a quote")
	// ErrQueryNone means user entered no quote
	ErrQueryNone = errors.New("no search terms entered")
)

/* Storer: quotes */

// quotes implements the Storer interface
type quotes struct {
	List []string
	Last int // THIS FIELD HAS BEEN DEPRECATED, DO NOT RELY ON IT, USE len(List) INSTEAD!
}

func (q *quotes) Index() string {
	return "quotes"
}

/* quote */

type quote struct {
	nilCommand
	Index []int `arg:"index"`
}

func newQuote() *quote { return &quote{} }

func (q *quote) Aliases() []string { return []string{"quote"} }

func (q *quote) Desc() string { return "Get a quote at given index. No args gives a random quote." }

func (q *quote) Subcommands() []commands.Command {
	return []commands.Command{
		newQuoteAdd(),
		newQuoteApprove(),
		newQuoteList(),
		newQuotePending(),
		newQuoteRemove(),
		newQuoteReject(),
		newQuoteSearch(),
		newQuoteClean(),
	}
}

func (q *quote) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	// Get quotes
	var quo quotes
	err := commands.DBGet(&quotes{}, keyQuotes, &quo)
	if err == commands.ErrDBNotFound {
		return nil, ErrQuoteEmpty
	} else if err != nil {
		return nil, err
	}

	// Check args
	var ind int
	if len(q.Index) == 0 {
		// Gen random number
		rand.Seed(time.Now().UnixNano())
		ind = rand.Intn(len(quo.List))
	} else {
		ind = q.Index[0]
		if ind >= len(quo.List) || ind < 0 {
			return nil, ErrQuoteIndex
		}
	}

	// Get quote and send it
	noMentions := utils.Unmention(ses, msg, quo.List[ind])
	return commands.NewSimpleSend(msg.ChannelID, noMentions), nil
}

type quoteAdd struct {
	nilCommand
	New []string `arg:"quote"`
}

func newQuoteAdd() *quoteAdd { return &quoteAdd{} }

func (q *quoteAdd) Aliases() []string { return []string{"quote add"} }

func (q *quoteAdd) Desc() string { return "Adds a quote to the pending list." }

func (q *quoteAdd) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	// Get the pending quote list from the db
	var pen quotes
	err := commands.DBGet(&quotes{}, keyPending, &pen)
	if err == commands.ErrDBNotFound {
		// Create a new quote list
		pen = quotes{
			List: []string{},
			Last: -1,
		}
	} else if err != nil {
		return nil, err
	}

	// Check quote first
	newQuote := strings.TrimSpace(strings.Join(q.New, " "))

	if len(newQuote) == 0 {
		// Quote is empty, throw error
		return nil, ErrQuoteNone
	}

	// Put the new quote into the pending quote list and update Last
	newQuote = strings.ReplaceAll(strings.Join(q.New, " "), `\n`, "\n")

	pen.List = append(pen.List, newQuote)
	//pen.Last++

	// Set the pending quote list in the db
	_, _, err = commands.DBSet(&pen, keyPending)
	if err != nil {
		return nil, err
	}

	// Send message to channel
	out := fmt.Sprintf("Added ```%s``` to the Pending list at index **#%d**", newQuote, len(pen.List)-1)
	return commands.NewSimpleSend(msg.ChannelID, out), nil
}

type quoteApprove struct {
	nilCommand
	Index int `arg:"index"`
}

func newQuoteApprove() *quoteApprove { return &quoteApprove{} }

func (q *quoteApprove) Aliases() []string { return []string{"quote approve", "quote ap"} }

func (q *quoteApprove) Desc() string { return "Approves a quote." }

func (q *quoteApprove) Roles() []string { return []string{"mod"} }

func (q *quoteApprove) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	// Get pending list
	var pen quotes
	err := commands.DBGet(&quotes{}, keyPending, &pen)
	if err == commands.ErrDBNotFound {
		return nil, ErrQuoteEmpty
	} else if err != nil {
		return nil, err
	}

	// Check index
	if err != nil || q.Index < 0 || q.Index >= len(pen.List) {
		return nil, ErrQuoteIndex
	}

	// Get approved list
	var quo quotes
	err = commands.DBGet(&quotes{}, keyQuotes, &quo)
	if err == commands.ErrDBNotFound {
		quo = quotes{
			List: []string{},
			Last: -1,
		}
	} else if err != nil {
		return nil, err
	}

	// Move pending quote to approved list, filling gaps first
	var ins int
	func() {
		if len(quo.List) == 0 {
			// quote list is empty
			quo.List = append(quo.List, pen.List[q.Index])
			ins = len(quo.List) - 1
			return
		}

		// quote list is not empty
		ins = len(quo.List)

		// find first empty index in the list
		for i, quote := range quo.List {
			if len(quote) == 0 {
				// found index, insert
				ins = i
				quo.List[ins] = pen.List[q.Index]
				return
			}
		}

		// didn't find index, insert at end
		quo.List = append(quo.List, pen.List[q.Index])
	}()

	// get all elements before the index
	newPen := pen.List[:q.Index]

	// not at the end, splice the rest on
	if q.Index != len(pen.List) {
		newPen = append(newPen, pen.List[q.Index+1:]...)
	}

	// set new pending list
	pen.List = newPen

	// Set quotes and pending
	_, _, err = commands.DBSet(&pen, keyPending)
	if err != nil {
		return nil, err
	}
	_, _, err = commands.DBSet(&quo, keyQuotes)
	if err != nil {
		return nil, err
	}

	out := fmt.Sprintf("Approved quote %s now at index **#%d**", utils.Block(quo.List[ins]), ins)

	return commands.NewSimpleSend(msg.ChannelID, out), nil
}

type quoteList struct {
	nilCommand
}

func newQuoteList() *quoteList { return &quoteList{} }

func (q *quoteList) Aliases() []string { return []string{"quote list", "quote ls"} }

func (q *quoteList) Desc() string {
	return fmt.Sprintf("Lists a range of approved quotes. Specify an index to look around it (defaults to %d).", quoteListLimit/2)
}

func (q *quoteList) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	// Get all approved quotes from db
	var quo quotes
	err := commands.DBGet(&quotes{}, keyQuotes, &quo)

	if err == commands.ErrDBNotFound {
		return nil, ErrQuoteEmpty
	} else if err != nil {
		return nil, err
	}

	timer := time.NewTimer(2 * time.Minute)
	go func() {
		// keep state of message
		page := 0
		// pages are indexed at 0
		// Ceil(30/15)-1 = 1		| 15 15
		// Ceil(31/15)-1 = 2 		| 15 15 1
		lastPage := int(math.Ceil(float64(len(quo.List))/float64(quoteListLimit))) - 1
		quoteLim := quoteListLimit
		once := false
		if len(quo.List) < quoteListLimit {
			lastPage = 0
			quoteLim = len(quo.List)
			once = true
		}

		// send a message first
		out := utils.Under("Quotes of UNSW PCSoc")
		for i, quote := range quo.List[0:quoteLim] {
			if quote != "" {
				out += fmt.Sprintf("\n**#%d:** %s", i, utils.Unmention(ses, msg, quote))
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
			return
		}

		err = ses.MessageReactionAdd(msg.ChannelID, outMessage.ID, emojiRight)
		if err != nil {
			return
		}

		unregister := ses.AddHandler(func(innerSes *discordgo.Session, event *discordgo.MessageReactionAdd) {
			reaction := event.MessageReaction

			// listen for reactions on the specific message sent
			if reaction.MessageID != outMessage.ID || reaction.UserID == outMessage.Author.ID {
				return
			}

			// ignore non-control emoji
			reactEmoji := reaction.Emoji.APIName()
			if reactEmoji != emojiLeft && reactEmoji != emojiRight {
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
				return
			}

			if reactEmoji == emojiLeft && page == 0 {
				page = lastPage
			} else if reactEmoji == emojiLeft {
				page--
			}

			if reactEmoji == emojiRight && page+1 > lastPage {
				page = 0
			} else if reactEmoji == emojiRight {
				page++
			}

			// calculate bounds
			left := page * quoteListLimit
			right := (page + 1) * quoteListLimit
			if right > len(quo.List) {
				right = len(quo.List)
			}

			// construct edit message
			edit := utils.Under("Quotes of UNSW PCSoc")
			for i, quote := range quo.List[left:right] {
				if quote != "" {
					edit += fmt.Sprintf("\n**#%d:** %s", i+left, utils.Unmention(ses, msg, quote))
				}
			}
			edit += fmt.Sprintf("\n`Page %d/%d`", page, lastPage)

			// actually edit the damn message
			innerSes.ChannelMessageEdit(reaction.ChannelID, reaction.MessageID, edit)

		})

		// wait until the timer is done, then unregister the handler and the reactions
		<-timer.C

		// yeet
		ses.MessageReactionsRemoveAll(msg.ChannelID, outMessage.ID)
		ses.MessageReactionsRemoveAll(msg.ChannelID, outMessage.ID)
		unregister()
	}()

	return nil, nil
}

type quotePending struct {
	nilCommand
	Index []int `arg:"index"`
}

func newQuotePending() *quotePending { return &quotePending{} }

func (q *quotePending) Aliases() []string { return []string{"quote pending", "quote pd"} }

func (q *quotePending) Desc() string { return "Lists all pending quotes." }

func (q *quotePending) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	// Get all pending quotes from db
	var pen quotes
	err := commands.DBGet(&quotes{}, keyPending, &pen)
	if err == commands.ErrDBNotFound {
		return nil, ErrQuoteEmpty
	} else if err != nil {
		return nil, err
	}

	// Check empty
	if len(pen.List) == 0 {
		return commands.NewSimpleSend(msg.ChannelID, "Pending list is empty."), nil
	}

	// Build output
	var out string
	if len(q.Index) == 0 {
		// List them
		out = utils.Under("Pending quotes:") + "\n"
		for i, q := range pen.List {
			out += utils.Bold("#"+strconv.Itoa(i)+":") + " " + q + "\n"
		}
	} else {
		ind := q.Index[0]
		// Check index
		if ind < 0 || ind >= len(pen.List) {
			return nil, ErrQuoteIndex
		}

		// TODO: test
		out = fmt.Sprintf("Pending quote at index **%d**:\n%s", q.Index[0], pen.List[ind])
	}

	return commands.NewSimpleSend(msg.ChannelID, out), nil
}

type quoteReject struct {
	nilCommand
	Index int `arg:"index"`
}

func newQuoteReject() *quoteReject { return &quoteReject{} }

func (q *quoteReject) Aliases() []string { return []string{"quote reject", "quote rj"} }

func (q *quoteReject) Desc() string { return "Rejects a quote from the pending list." }

func (q *quoteReject) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	// Get pending list
	var pen quotes
	err := commands.DBGet(&quotes{}, keyPending, &pen)
	if err == commands.ErrDBNotFound {
		return nil, ErrQuoteEmpty
	} else if err != nil {
		return nil, err
	}

	// Check index
	if q.Index < 0 || q.Index >= len(pen.List) {
		return nil, ErrQuoteIndex
	}

	// Reorder list
	rej := pen.List[q.Index]
	newPen := pen.List[:q.Index]
	if q.Index != len(pen.List) {
		newPen = append(newPen, pen.List[q.Index+1:]...)
	}
	pen.List = newPen
	//pen.Last--

	// Set pending
	_, _, err = commands.DBSet(&pen, keyPending)
	if err != nil {
		return nil, err
	}

	out := "Rejected quote\n" + utils.Block(rej)
	return commands.NewSimpleSend(msg.ChannelID, out), nil
}

type quoteRemove struct {
	nilCommand
	Index int `arg:"index"`
}

func newQuoteRemove() *quoteRemove { return &quoteRemove{} }

func (q *quoteRemove) Aliases() []string { return []string{"quote remove", "quote rm"} }

func (q *quoteRemove) Desc() string { return "Removes a quote." }

func (q *quoteRemove) Roles() []string { return []string{"mod"} }

func (q *quoteRemove) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	// Get quotes list
	var quo quotes
	err := commands.DBGet(&quotes{}, keyQuotes, &quo)
	if err == commands.ErrDBNotFound {
		return nil, ErrQuoteEmpty
	} else if err != nil {
		return nil, err
	}

	// Check index
	if q.Index < 0 || q.Index >= len(quo.List) {
		return nil, ErrQuoteIndex
	}

	// Clear quote at index, don't reorder
	rem := quo.List[q.Index]
	quo.List[q.Index] = ""

	// Set quotes
	_, _, err = commands.DBSet(&quo, keyQuotes)
	if err != nil {
		return nil, err
	}

	out := "Removed quote\n" + utils.Block(rem)
	return commands.NewSimpleSend(msg.ChannelID, out), nil
}

type quoteSearch struct {
	nilCommand
	Query []string `arg:"query"`
}

type searchMatch struct {
	content string
	index   int
}

func newQuoteSearch() *quoteSearch { return &quoteSearch{} }

func (q *quoteSearch) Aliases() []string { return []string{"quote search", "quote se"} }

func (q *quoteSearch) Desc() string {
	return fmt.Sprintf("Searches for a quote, returns top %d results.", searchLimit)
}

func (q *quoteSearch) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	// Join query
	qry := strings.TrimSpace(strings.Join(q.Query, "[ \\._-]*"))

	// Enforce only alphanumeric regex
	qry = regexp.MustCompile("[^a-zA-Z0-9]+").ReplaceAllString(qry, "")
	if len(qry) == 0 {
		return nil, ErrQueryNone
	}

	// Get quotes list
	var quo quotes
	err := commands.DBGet(&quotes{}, keyQuotes, &quo)
	if err == commands.ErrDBNotFound {
		return nil, ErrQuoteEmpty
	} else if err != nil {
		return nil, err
	}

	matches := []*searchMatch{}
	for i, quote := range quo.List {
		match, err := regexp.Match("(?i)"+qry, []byte(quote))
		if err != nil {
			return nil, err
		}

		if match {
			matches = append(matches, &searchMatch{
				content: quote,
				index:   i,
			})
		}
	}

	if len(matches) == 0 {
		return commands.NewSimpleSend(msg.ChannelID, "No matches found."), nil
	}

	// print results
	out := "Search Results:\n"
	for i, match := range matches {
		if i == searchLimit {
			break
		}
		out += utils.Bold("#"+strconv.Itoa(match.index)+": ") + match.content + "\n"
	}

	return commands.NewSimpleSend(msg.ChannelID, out), nil
}

type quoteClean struct {
	nilCommand
}

func newQuoteClean() *quoteClean { return &quoteClean{} }

func (q *quoteClean) Aliases() []string { return []string{"quote clean", "quote cl"} }

func (q *quoteClean) Desc() string { return "Replaces `\\n` characters with newlines." }

func (q *quoteClean) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	// Get quotes list
	var quo quotes
	err := commands.DBGet(&quotes{}, keyQuotes, &quo)
	if err == commands.ErrDBNotFound {
		return nil, ErrQuoteEmpty
	} else if err != nil {
		return nil, err
	}

	for i := 0; i < len(quo.List); i++ {
		quo.List[i] = strings.ReplaceAll(quo.List[i], `\n`, "\n")
	}

	// Set quotes
	_, _, err = commands.DBSet(&quo, keyQuotes)
	if err != nil {
		return nil, err
	}

	return commands.NewSimpleSend(msg.ChannelID, "All Clean! ✨"), nil
}
