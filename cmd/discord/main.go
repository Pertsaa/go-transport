package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var (
	token string
	vcs   = make(map[string]*discordgo.VoiceConnection)
	vcMu  sync.Mutex
)

var commands = []*discordgo.ApplicationCommand{
	{Name: "stream", Description: "Start streaming"},
	{Name: "stop", Description: "Stop streaming"},
}

func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.Parse()
}

func main() {
	if token == "" {
		fmt.Println("No token provided. Please run: bot -t <bot token>")
		return
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	dg.AddHandler(ready)
	dg.AddHandler(interactionCreate)
	dg.AddHandler(guildCreate)
	dg.AddHandler(voiceStateUpdate)

	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildVoiceStates

	if err := dg.Open(); err != nil {
		fmt.Println("Error opening Discord session: ", err)
		return
	}

	for _, v := range commands {
		_, err := dg.ApplicationCommandCreate(dg.State.User.ID, "", v)
		if err != nil {
			log.Println("Cannot create slash command:", err)
		}
	}

	fmt.Println("stream bot is running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	cleanup()
	dg.Close()
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateGameStatus(0, "/stream")
}

func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cmd := i.ApplicationCommandData().Name

	if cmd == "stream" {
		streamInteraction(s, i)
		return
	}

	if cmd == "stop" {
		stopInteraction(s, i)
		return
	}
}

// Join user vc and start streaming
func streamInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	joinUserVoiceChannel(s, i)

	go streamAudio(i.GuildID)

	respondMessage(s, i, "Streaming...")
}

// Stop streaming and disconnect vc
func stopInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	vcMu.Lock()
	defer vcMu.Unlock()
	if vc, ok := vcs[i.GuildID]; ok {
		vc.Disconnect()
		delete(vcs, i.GuildID)
	}

	respondMessage(s, i, "Stopped streaming.")
}

func voiceStateUpdate(s *discordgo.Session, vs *discordgo.VoiceStateUpdate) {
	guildID := vs.GuildID
	vc, ok := vcs[guildID]
	if !ok {
		return
	}

	guild, err := s.State.Guild(guildID)
	if err != nil {
		return
	}

	userCount := 0
	for _, vs := range guild.VoiceStates {
		if vs.ChannelID == vc.ChannelID && vs.UserID != s.State.User.ID {
			userCount++
		}
	}

	if userCount == 0 {
		vcMu.Lock()
		defer vcMu.Unlock()
		if vc, ok := vcs[guildID]; ok {
			vc.Disconnect()
			delete(vcs, guildID)
		}
		log.Println("Disconnected from empty voice channel in guild:", guildID)
	}
}

func streamAudio(guildID string) error {
	vc, ok := vcs[guildID]
	if !ok {
		return fmt.Errorf("vc not found for guild %s", guildID)
	}
	vc.Speaking(true)
	defer vc.Speaking(false)

	// vc.OpusSend <- opusFrame

	return nil
}

func joinUserVoiceChannel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	g, err := s.State.Guild(i.GuildID)
	if err != nil {
		return
	}

	var userChannelID string
	for _, vs := range g.VoiceStates {
		if vs.UserID == i.Member.User.ID {
			userChannelID = vs.ChannelID
			break
		}
	}

	if userChannelID == "" {
		respondMessage(s, i, "Join a voice channel first!")
		return
	}

	vc, err := s.ChannelVoiceJoin(i.GuildID, userChannelID, false, true)
	if err != nil {
		respondMessage(s, i, "Failed to join voice channel.")
		return
	}

	vcMu.Lock()
	vcs[i.GuildID] = vc
	vcMu.Unlock()
}

func respondMessage(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: message},
	})
}

func guildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild.Unavailable {
		return
	}
	for _, channel := range event.Guild.Channels {
		if channel.ID == event.Guild.ID {
			s.ChannelMessageSend(channel.ID, "stream is ready! Use /stream while in a voice channel to play a stream.")
			return
		}
	}
}

func cleanup() {
	for _, vc := range vcs {
		vc.Disconnect()
	}

	vcMu.Lock()
	vcs = make(map[string]*discordgo.VoiceConnection)
	vcMu.Unlock()
}

// func connectWS() {
// 	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"}
// 	log.Printf("Connecting to %s", u.String())

// 	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
// 	if err != nil {
// 		log.Fatal("dial:", err)
// 	}
// 	defer c.Close()
// }
