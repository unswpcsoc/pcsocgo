package handlers

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sahilm/fuzzy"

	"github.com/unswpcsoc/pcsocgo/commands"
	"github.com/unswpcsoc/pcsocgo/internal/utils"
)

const (
	keyPending = "pending"
	keyQuotes  = "approve"

	quoteLineLimit = 80
	quoteListLimit = 10
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
		if ind > len(quo.List) || ind < 0 {
			return nil, ErrQuoteIndex
		}
	}

	// Get quote and send it
	return commands.NewSimpleSend(msg.ChannelID, quo.List[ind]), nil
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
	if err != nil || q.Index < 0 || q.Index > len(pen.List) {
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
			//quo.Last++
			ins = len(quo.List)
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
		//quo.Last = ins
	}()

	// get all elements before the index
	newPen := pen.List[:q.Index]

	// not at the end, splice the rest on
	if q.Index != len(pen.List) {
		newPen = append(newPen, pen.List[q.Index+1:]...)
	}

	// set new pending list
	pen.List = newPen
	//pen.Last--

	// Set quotes and pending
	_, _, err = commands.DBSet(&pen, keyPending)
	if err != nil {
		return nil, err
	}
	_, _, err = commands.DBSet(&quo, keyQuotes)
	if err != nil {
		return nil, err
	}

	out := fmt.Sprintf("Approved quote ```%s``` now at index **#%d**", utils.Block(quo.List[ins]), ins)

	return commands.NewSimpleSend(msg.ChannelID, out), nil
}

type quoteList struct {
	nilCommand
	Index []int `arg:"lookaround"`
}

func newQuoteList() *quoteList { return &quoteList{} }

func (q *quoteList) Aliases() []string { return []string{"quote list", "quote ls"} }

func (q *quoteList) Desc() string {
	return "Lists a range of approved quotes. Specify an index to look around it (defaults to 10)."
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

	ind := 10
	if len(quo.List) < ind {
		ind = 0
	}

	// check if user specified an index
	if len(q.Index) > 0 {
		// Check index
		if q.Index[0] < 0 || q.Index[0] >= len(quo.List) {
			return nil, ErrQuoteIndex
		}
		ind = q.Index[0]
	}

	// closure so we don't have to repeat this logic
	appendQuote := func(buf string, i int, quote string) string {
		// deleted quote, skip
		if len(quote) == 0 {
			return buf
		}

		if len(quote) > quoteLineLimit {
			quote = quote[:quoteLineLimit] + "[...]"
		}

		// don't worry about message limit, won't be reached
		buf += fmt.Sprintf("**#%d:** %s\n", i, quote)
		return buf
	}

	// List away!
	before := ind - (quoteListLimit / 2)
	if before < 0 {
		before = 0
	}

	after := ind + (quoteListLimit / 2) + 1
	if after > len(quo.List) {
		after = len(quo.List)
	}

	var out = fmt.Sprintf("__There are %d Quotes, displaying quotes from index %d to %d:__\n", len(quo.List), before, after-1)

	for i := before; i < ind; i++ {
		out = appendQuote(out, i, quo.List[i])
	}

	for i := ind; i < after; i++ {
		out = appendQuote(out, i, quo.List[i])
	}

	return commands.NewSimpleSend(msg.ChannelID, out), nil
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

func newQuoteSearch() *quoteSearch { return &quoteSearch{} }

func (q *quoteSearch) Aliases() []string { return []string{"quote search", "quote se"} }

func (q *quoteSearch) Desc() string { return "Searches for a quote, returns top 5 results." }

func (q *quoteSearch) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	// Join query
	qry := strings.TrimSpace(strings.Join(q.Query, " "))
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

	// use fuzzy finding to get top 5 results
	// TODO: find a better fuzzy find
	mat := fuzzy.Find(qry, quo.List)
	if len(mat) == 0 {
		return commands.NewSimpleSend(msg.ChannelID, "No matches found."), nil
	}

	// print results
	out := "Search Results:\n"
	for i, m := range mat {
		if i == 5 {
			break
		}
		out += utils.Bold("#"+strconv.Itoa(m.Index)+": ") + m.Str + "\n"
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