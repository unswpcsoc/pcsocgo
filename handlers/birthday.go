package handlers

import (
	"errors"
	logs "log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/unswpcsoc/pcsocgo/commands"
)

const (
	bdaysKey = "birthdays"
)

type birthdayStorer struct {
	Birthdays map[string]time.Time
}

func (b *birthdayStorer) Index() string {
	return "birthday"
}

type Birthday struct {
	nilCommand
	Birthday string `arg:"birthday"`
}

func newBirthday() *Birthday { return &Birthday{} }

func (b *Birthday) Aliases() []string { return []string{"bday", "birthday", "birthday add", "bday add"} }

func (b *Birthday) Desc() string {
	return "Adds your birthday to the bot, will give you the role on the date provided. Format must be `2/jan`"
}

func (b *Birthday) Subcommands() []commands.Command { return []commands.Command{newBirthdayRemove()} }

func (b *Birthday) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	// parse the birthday
	bdayString := strings.Trim(b.Birthday, " ")

	birthday, err := time.Parse("2/Jan/2006", bdayString+"/2006")
	if err != nil {
		return nil, err
	}

	// get database
	var bdays birthdayStorer
	err = commands.DBGet(&bdays, bdaysKey, &bdays)
	if err == commands.ErrDBNotFound {
		// create entry
		bdays = birthdayStorer{make(map[string]time.Time)}
	} else if err != nil {
		return nil, err
	}

	if bdays.Birthdays == nil {
		bdays.Birthdays = make(map[string]time.Time)
	}

	bdays.Birthdays[msg.Author.ID] = birthday

	// set in database
	commands.DBLock()
	defer commands.DBUnlock()

	_, _, err = commands.DBSet(&bdays, bdaysKey)
	if err != nil {
		return nil, err
	}

	return commands.NewSimpleSend(msg.ChannelID, "Added your birthday "+bdayString), nil
}

type BirthdayRemove struct {
	nilCommand
}

func newBirthdayRemove() *BirthdayRemove { return &BirthdayRemove{} }

func (b *BirthdayRemove) Aliases() []string {
	return []string{"bday remove", "bday rm", "birthday remove", "birthday rm"}
}

func (b *BirthdayRemove) Desc() string {
	return "Removes your birthday from the bot"
}

func (b *BirthdayRemove) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	// get database
	var bdays birthdayStorer
	commands.DBGet(&bdays, bdaysKey, &bdays)

	delete(bdays.Birthdays, msg.Author.ID)

	// set in database
	commands.DBLock()
	defer commands.DBUnlock()

	commands.DBSet(&bdays, bdaysKey)

	return commands.NewSimpleSend(msg.ChannelID, "Removed your birthday"), nil
}

type BirthdayModCheck struct {
	nilCommand
}

func newBirthdayModCheck() *BirthdayModCheck { return &BirthdayModCheck{} }

func (b *BirthdayModCheck) Aliases() []string {
	return []string{"bday check", "bday modcheck", "birthday modcheck", "birthday check"}
}

func (b *BirthdayModCheck) Desc() string {
	return "Mod utility to check and give the birthday roles manually"
}

func (b *BirthdayModCheck) Roles() []string { return []string{"mod", "exec"} }

func (b *BirthdayModCheck) MsgHandle(ses *discordgo.Session, msg *discordgo.Message) (*commands.CommandSend, error) {
	location, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		return nil, errors.New("location is bad :(")
	}

	now := time.Now()
	aestTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), location)
	err = doBirthday(ses, aestTime)
	if err != nil {
		return nil, err
	}

	return commands.NewSimpleSend(msg.ChannelID, "Check complete!"), nil
}

func doBirthday(ses *discordgo.Session, tim time.Time) error {
	// call handler
	logs.Println("Calling birthday handler for time:", tim)

	var bdays birthdayStorer
	err := commands.DBGet(&bdays, bdaysKey, &bdays)
	if err == commands.ErrDBNotFound {
		logs.Println("No birthdays found in db")
		return err
	}

	// get birthday role
	guildroles, err := ses.GuildRoles(commands.Guild.ID)
	if err != nil {
		return err
	}

	roleID := ""
	for _, role := range guildroles {
		if strings.Contains(strings.ToLower(role.Name), strings.ToLower("birthday")) {
			roleID = role.ID
			break
		}
	}

	if len(roleID) == 0 {
		return errors.New("no birthday role in guild: " + commands.Guild.Name + "\n")
	}

	// iterate birthdays
	for uid, bday := range bdays.Birthdays {
		_, err := ses.User(uid)
		if err != nil {
			logs.Println("Could not find user with id:", uid)
		}

		// check that the day is right
		if !(bday.Month() == tim.Month() && bday.Day() == tim.Day()) {
			_ = ses.GuildMemberRoleRemove(commands.Guild.ID, uid, roleID)
			continue
		}

		// HAPPY @Birthday!
		err = ses.GuildMemberRoleAdd(commands.Guild.ID, uid, roleID)
		if err != nil {
			logs.Println(err)
			continue
		}
	}

	return nil
}

func initBirthday(ses *discordgo.Session) chan bool {
	logs.Println("Initialised birthday")

	location, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		panic(errors.New("location is bad :("))
	}

	ticker := time.NewTicker(time.Minute)
	done := make(chan bool)

	now := time.Now()
	aestTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), location)

	go func() {
		// handle done signal
		select {
		case <-done:
			return
		default:
			// check at aest midnight
			if !(aestTime.Hour() == 0 && aestTime.Minute() == 0) {
				return
			}
			err := doBirthday(ses, aestTime)
			if err != nil {
				logs.Println("birthDaemon:", err)
			}
		}

		// handle ticker
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				// check at aest midnight
				if !(aestTime.Hour() == 0 && aestTime.Minute() == 0) {
					continue
				}
				err := doBirthday(ses, aestTime)
				if err != nil {
					logs.Println("birthDaemon:", err)
				}
			}
		}
	}()
	return done
}
