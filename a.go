package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var (
	userToken string
)

func main() {
	godotenv.Load()

	userToken = os.Getenv("DISCORD_TOKEN")

	if userToken == "" {
		fmt.Println("Please set DISCORD_TOKEN environment variable")
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

	if m.Content == ",leaveall" {
		s.ChannelMessageDelete(m.ChannelID, m.ID)
		fmt.Println("Leaving all servers...")

		guilds, err := s.UserGuilds(100, "", "", false)
		if err != nil {
			fmt.Println("Error getting guilds:", err)
		} else {
			for _, g := range guilds {
				err := s.GuildLeave(g.ID)
				if err != nil {
					fmt.Printf("Error leaving %s: %v\n", g.Name, err)
				} else {
					fmt.Printf("Left server: %s\n", g.Name)
				}
				time.Sleep(1 * time.Second)
			}
		}
		fmt.Println("Finished leaving servers")
	}

	if m.Content == ",unfriendall" {
		s.ChannelMessageDelete(m.ChannelID, m.ID)
		fmt.Println("Removing all friends...")

		type Relationship struct {
			ID   string `json:"id"`
			Type int    `json:"type"`
			User struct {
				ID       string `json:"id"`
				Username string `json:"username"`
			} `json:"user"`
		}

		var rels []Relationship
		data, err := s.Request("GET", "/users/@me/relationships", nil)
		if err != nil {
			fmt.Println("Error getting relationships:", err)
		} else {
			err = json.Unmarshal(data, &rels)
			if err != nil {
				fmt.Println("Error parsing relationships:", err)
			} else {
				for _, r := range rels {
					if r.Type == 1 {
						_, err := s.Request("DELETE", "/users/@me/relationships/"+r.ID, nil)
						if err != nil {
							fmt.Printf("Error removing friend %s: %v\n", r.User.Username, err)
						} else {
							fmt.Printf("Removed friend: %s\n", r.User.Username)
						}
						time.Sleep(1 * time.Second)
					}
				}
			}
		}
		fmt.Println("Finished removing friends")
	}

}
