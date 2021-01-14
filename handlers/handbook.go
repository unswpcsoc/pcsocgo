package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"html"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"github.com/unswpcsoc/pcsocgo/commands"
	"github.com/bwmarrin/discordgo"
	"github.com/microcosm-cc/bluemonday"
)

var (
	ErrNoInput        = errors.New("No Course code entered")
	ErrInvalidFormat  = errors.New("Invalid Course Code Format")
	ErrNotFound       = errors.New("Course Not Found")
	ErrScrapingFailed = errors.New("Web scraping failed, please contact an Exec")
)

type handbook struct {
	nilCommand
	Code string `arg:"code"`
}

type Body struct {
	Contentlets []Contentlets
}

type Contentlets struct {
	Data string
	Urlmap string
}

type Data struct {
	Title string
	Description string
	Enrolment_Rules []Enrolment_Rules
	Offering_Detail Offering_Detail
}

type Enrolment_Rules struct {
	Description string
}

type Offering_Detail struct {
	Offering_Terms string
}


func newHandbook() *handbook { return &handbook{} }

func (h *handbook) Aliases() []string { return []string{"handbook"} }

func (h *handbook) Desc() string { return "Searches handbook.unsw for course" }

func (h *handbook) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {

	// Special case for DELL1234
	if strings.ToUpper(h.Code) == "DELL1234" {
		message := commands.NewSend(msg.ChannelID)
		embed := makeMessage("https://webapps.cse.unsw.edu.au/webcms2/course/index.php?cid=1137",
			"How to blow up my computer?",
			"No Course Outline (yet)",
			"None",
			"None")
		return message.Embed(embed), nil
	}

	// Check course code
	match, err := regexp.MatchString(`^[A-Za-z]{4}[0-9]{4}$`, h.Code)
	if err != nil {
		return nil, err
	}
	if !match {
		return nil, ErrInvalidFormat
	}

	// get data with magic function
	var url, title, desc, term, cond = "None", "None", "None", "None", "None"
	url, title, desc, term, cond, err = postSearch(strings.ToUpper(h.Code), "undergraduate")
	if err != nil {
		url, title, desc, term, cond, err = postSearch(strings.ToUpper(h.Code), "postgraduate")
	}
	if err != nil {
		return nil, err
	}

	// create and send message
	message := commands.NewSend(msg.ChannelID)
	embed := makeMessage(url, title, desc, term, cond)
	return message.Embed(embed), nil
}

func postSearch(Code string, Graduate string) (url string, title string, desc string, term string, cond string, err error) {
	//establish post request
	posturl := "https://www.handbook.unsw.edu.au/api/es/search"
	var requestJson = []byte(`{"query":{"bool":{"must":[{"query_string":{"query":"unsw_psubject.code: ` + Code + `"}},{"term":{"live":true}},{"bool":{"minimum_should_match":"100%","should":[{"query_string":{"fields":["unsw_psubject.studyLevelURL"],"query":"` + Graduate + `"}}]}}]}},"aggs":{"implementationYear":{"terms":{"field":"unsw_psubject.implementationYear_dotraw","size":100}},"availableInYears":{"terms":{"field":"unsw_psubject.availableInYears_dotraw","size":100}}},"size":100,"_source":{"includes":["versionNumber","availableInYears","implementationYear"]}}`)
	req, err := http.NewRequest("POST", posturl, bytes.NewBuffer(requestJson))
	req.Header.Set("Content-Type", "application/json")

	//perform post
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", "", "", ErrScrapingFailed
	}
	defer resp.Body.Close()

	//read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", "", "", ErrScrapingFailed
	}
	var bodyJs Body
	json.Unmarshal(body, &bodyJs)
	if len(bodyJs.Contentlets) == 0 {
		return "", "", "", "", "", ErrNotFound
	}

	//search response for latest year
	yearNo := 0
	for i := range bodyJs.Contentlets{
		if bodyJs.Contentlets[yearNo].Urlmap < bodyJs.Contentlets[i].Urlmap {
			yearNo = i
		}
	}

	url = "https://www.handbook.unsw.edu.au" + bodyJs.Contentlets[yearNo].Urlmap

	//parse "data" field of response body as json to extract relevant fields
	var data Data
	json.Unmarshal([]byte(bodyJs.Contentlets[yearNo].Data), &data)
	p := bluemonday.StripTagsPolicy()
	desc = p.Sanitize(data.Description)
	desc = html.UnescapeString(desc)

	term = "None"
	if data.Offering_Detail.Offering_Terms != "" {
		term = data.Offering_Detail.Offering_Terms
	}

	cond = "None"
	if len(data.Enrolment_Rules) > 0 {
		// cut <br> at the end of many conditions. Otherwise does nothing
		cond = strings.Split(data.Enrolment_Rules[0].Description, "<")[0]
	}

	return url, data.Title, desc, term, cond, nil
}

func makeMessage(Url string, Title string, Desc string, Term string, Cond string) *discordgo.MessageEmbed {
	// create fields for embed
	terms := discordgo.MessageEmbedField{
		Name:   "Offering Terms",
		Value:  Term,
		Inline: true,
	}
	conds := discordgo.MessageEmbedField{
		Name:   "Enrolment Conditions",
		Value:  Cond,
		Inline: true,
	}
	messagefields := []*discordgo.MessageEmbedField{&terms, &conds}
	if len(Desc) > 1024 {
		Desc = Desc[:1024] + "..."
	}
	// make embed and return
	embed := &discordgo.MessageEmbed{
		URL:         Url,
		Title:       Title,
		Description: Desc,
		Fields:      messagefields,
		Color:       0xFDD600,
	}

	return embed
}
