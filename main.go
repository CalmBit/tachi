package main

import (
	"database/sql"
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

// Suggested perms: 536995904

var token string
var debug bool
var db *sql.DB

var adminRoles = make(map[string][]string, 0)

// Skeleton borrowed from airhorn example at
// https://github.com/bwmarrin/discordgo/blob/master/examples/airhorn/main.go
func init() {
	flag.StringVar(&token, "t", "", "Token for discord")
	flag.BoolVar(&debug, "d", false, "Debug?")
	flag.Parse()

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	var err error
	db, err = sql.Open("sqlite3", "./data.db")

	if err != nil {
		log.Errorf("Unable to get SQLite connection - %s", err)
		os.Exit(-1)
	}

	_, err = db.Exec(
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			token TEXT NOT NULL
		);`)

	if err != nil {
		log.Errorf("Bad create on `users` - %s", err)
		os.Exit(-1)
	}

	_, err = db.Exec(
		`CREATE TABLE IF NOT EXISTS systems (
			id TEXT PRIMARY KEY,
			name TEXT,
			description TEXT,
			tag TEXT,
			avatar_url TEXT,
			tz TEXT,
			description_privacy TEXT,
			member_list_privacy TEXT,
			front_privacy TEXT,
			front_history_privacy TEXT
		);`)

	if err != nil {
		log.Errorf("Bad create on `systems` - %s", err)
		os.Exit(-1)
	}
}

func main() {
	if token == "" {
		ex, err := os.Executable()
		if err == nil {
			log.Errorf("no token provided - run `%s -t <token>`\n", ex)
		} else {
			log.Error("no token provided - run with `-t <token>`\n")
		}
		os.Exit(-1)
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Errorf("Error creating discord session: %s\n", err)
		os.Exit(-1)
	}

	dg.AddHandler(onReady)
	dg.AddHandler(onGuildCreate)
	dg.AddHandler(onGuildRoleUpdate)
	dg.AddHandler(onMessage)

	//dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMembers | discordgo.IntentsGuildEmojis | discordgo.IntentsGuildWebhooks | discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions)

	err = dg.Open()
	if err != nil {
		log.Errorf("Error opening discord session: %s\n", err)
		os.Exit(-1)
	}

	log.Info("tachi should be running!")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}

func onReady(s *discordgo.Session, r *discordgo.Ready) {
	log.Debug("running ready routine\n")
	log.Debug("finished ready routine\n")
}

func onGuildRoleUpdate(s *discordgo.Session, r *discordgo.GuildRoleUpdate) {
	log.Debugf("detected role change for guild ID %s\n", r.GuildID)
	authenticated := rolePermTest(r.Role)
	if contains(adminRoles[r.GuildID], r.Role.ID) && !authenticated {
		log.Debugf("removing admin privs of role %s guild ID %s\n", r.Role.ID, r.GuildID)
		i := indexOf(adminRoles[r.GuildID], r.Role.ID)
		adminRoles[r.GuildID] = remove(adminRoles[r.GuildID], i)
	} else if authenticated {
		log.Debugf("adding role with id %s on GID %s\n", r.Role.ID, r.GuildID)
		adminRoles[r.GuildID] = append(adminRoles[r.GuildID], r.Role.ID)
	}
}

func onGuildCreate(s *discordgo.Session, r *discordgo.GuildCreate) {
	log.Debugf("detecting admin roles for guild ID %s\n", r.Guild.ID)
	if adminRoles[r.Guild.ID] == nil {
		adminRoles[r.Guild.ID] = make([]string, 0)
	}
	for _, role := range r.Guild.Roles {
		log.Debugf("testing role %s for guild ID %s\n", role.ID, r.Guild.ID)
		if rolePermTest(role) {
			log.Debugf("adding role with id %s on GID %s\n", role.ID, r.Guild.ID)
			adminRoles[r.Guild.ID] = append(adminRoles[r.Guild.ID], role.ID)
		}
	}
}

func onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	authenticated := false

	for _, r := range m.Member.Roles {
		if contains(adminRoles[m.GuildID], r) {
			authenticated = true
			break
		}
	}

	if !authenticated {
		log.Warn(os.Stderr, "User not able to use bot\n")
		return
	}

	_, err := s.ChannelMessageSend(m.ChannelID, strings.Join(m.Member.Roles, ","))

	if err != nil {
		log.Error("Unable to send message: %s\n", err)
	}
}

func rolePermTest(r *discordgo.Role) bool {
	return (r.Permissions&discordgo.PermissionManageMessages != 0) || (r.Permissions&discordgo.PermissionAdministrator != 0)
}
