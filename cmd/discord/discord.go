package main

import (
	"errors"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/christopher-dG/gosubscribe"
)

var (
	testChannel = os.Getenv("DISCORD_TEST_CHANNEL")
	test        = os.Getenv("DB_NAME") == "test"
	me          = os.Getenv("DISCORD_ME")
)

func main() {
	gosubscribe.Connect(
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PASS"),
	)

	discord, err := discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	discord.AddHandler(handleMessage)

	err = discord.Open()
	if err != nil {
		log.Fatal(err)
	}
	err = discord.UpdateStatus(0, "osu! as Slow Twitch")
	if err != nil {
		log.Println(err)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	err = discord.Close()
	if err != nil {
		log.Fatal(err)
	}
}

// handleMessage handles incoming messages.
func handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if isPrivate(s, m) {
		go handlePrivate(s, m)
	} else {
		go handlePublic(s, m)
	}
}

// isPrivate determines whether or not an incoming message is in a private channel.
func isPrivate(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	channel, err := s.State.Channel(m.ChannelID)
	if err != nil {
		return true // TBD.
	}
	return channel.Type == discordgo.ChannelTypeDM
}

// escape escapes Markdown underline formatting in strings.
func escape(s string) string {
	return strings.Replace(s, "_", "\\_", -1)
}

// getUser retrieves a user from the database.
func getUser(dUser *discordgo.User) (*gosubscribe.User, error) {
	user := new(gosubscribe.User)
	gosubscribe.DB.Where("discord_id = ?", dUser.ID).Find(user)
	if user.ID == 0 {
		return user, errors.New("not initialized")
	}
	return user, nil
}

// createUser adds a new user to the database if they don't already exist, and registers
// their Discord ID to their account.
func createUser(dUser *discordgo.User) (*gosubscribe.User, error) {
	_, err := getUser(dUser)
	if err == nil {
		return nil, errors.New("already initialized")
	}
	user := new(gosubscribe.User)
	id, _ := strconv.ParseInt(dUser.ID, 10, 64)
	user.DiscordID.Int64 = id
	user.DiscordID.Valid = true
	user.Secret = gosubscribe.GenSecret()
	gosubscribe.DB.Save(user)
	return user, nil
}
