package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var (
	userToken string
	channelID string
	isRunning bool
	mu        sync.Mutex
	stopChan  chan struct{}
	delayTime int64 = 15
	fishCount int64 = 0
)

func main() {
	godotenv.Load()

	userToken = os.Getenv("DISCORD_TOKEN")
	channelID = os.Getenv("CHANNEL_ID")

	if userToken == "" || channelID == "" {
		fmt.Println("Please set DISCORD_TOKEN and CHANNEL_ID environment variables")
		os.Exit(1)
	}

	dg, err := discordgo.New(userToken)
	if err != nil {
		fmt.Println("Error creating Discord session:", err)
		os.Exit(1)
	}

	dg.AddHandler(messageCreate)

	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening Discord connection:", err)
		os.Exit(1)
	}

	fmt.Println("Press Ctrl+C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID != s.State.User.ID {
		return
	}

	if m.Content == ",start" {
		mu.Lock()
		if isRunning {
			mu.Unlock()
			fmt.Println("Fishing already running")
			return
		}

		isRunning = true
		stopChan = make(chan struct{})
		mu.Unlock()

		s.ChannelMessageDelete(m.ChannelID, m.ID)
		fmt.Println("Fishing started")
		go fishMessages(s)
	}

	if m.Content == ",stop" {
		mu.Lock()
		if !isRunning {
			mu.Unlock()
			fmt.Println("Fishing not running")
			return
		}

		isRunning = false
		close(stopChan)
		mu.Unlock()

		s.ChannelMessageDelete(m.ChannelID, m.ID)
		fmt.Println("Fishing stopped")
	}

	if strings.HasPrefix(m.Content, ",delay ") {
		delayStr := strings.TrimPrefix(m.Content, ",delay ")
		parsedDelay, err := strconv.ParseInt(delayStr, 10, 64)
		if err != nil || parsedDelay <= 0 {
			fmt.Println("Invalid delay. Use: ,delay <seconds>")
			return
		}

		mu.Lock()
		delayTime = parsedDelay
		mu.Unlock()

		s.ChannelMessageDelete(m.ChannelID, m.ID)
		fmt.Printf("Delay set to %d seconds\n", parsedDelay)
	}
}

func fishMessages(s *discordgo.Session) {
	statsTicker := time.NewTicker(30 * time.Minute)
	defer statsTicker.Stop()

	for {
		mu.Lock()
		currentDelay := delayTime
		mu.Unlock()

		ticker := time.NewTicker(time.Duration(currentDelay) * time.Second)
		defer ticker.Stop()

		select {
		case <-stopChan:
			return
		case <-statsTicker.C:
			mu.Lock()
			count := fishCount
			mu.Unlock()
			fmt.Printf("=== 30 MIN UPDATE === Messages sent: %d\n", count)
		case <-ticker.C:
			_, err := s.ChannelMessageSend(channelID, "fish")
			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			mu.Lock()
			fishCount++
			mu.Unlock()
		}
	}
}
