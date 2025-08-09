package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func tryConnect(dg *discordgo.Session) {
	for {
		err := dg.Open()
		if err == nil {
			return
		}
		log.Printf("Error connecting: %s, retrying...", err)
		time.Sleep(5 * time.Second)
	}
}

func getLinks(link string) (string, error) {
	cmd := exec.Command("yt-dlp", "-S", "proto", "--proxy", config.Proxy, "--get-url", link)
	log.Printf("Running: %s", cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	result := make(chan string)
	errors := make(chan error)
	err = cmd.Start()
	if err != nil {
		return "", err
	}
	go func() {
		r, err := io.ReadAll(stderr)
		if err != nil {
			result <- err.Error()
		}
		if len(r) == 0 {
			return
		}
		errors <- fmt.Errorf("%s", r)
	}()
	go func() {
		r, err := io.ReadAll(stdout)
		if err != nil {
			result <- err.Error()
		}
		if len(r) == 0 {
			return
		}
		result <- string(r)
	}()
	timeout := time.After(time.Second * 30)
	select {
	case r := <-result:
		if strings.HasPrefix(r, "http") {
			lines := strings.Split(r, "\n")
			return lines[0], nil
		}
		return "", fmt.Errorf("invalid return value: '%s'", r)
	case e := <-errors:
		return "", e
	case <-timeout:
		cmd.Process.Kill()
		return "", fmt.Errorf("timed out")
	}
}

func main() {
	var dg *discordgo.Session
	var err error
	loadConfig("config.json")
	dg, err = discordgo.New("Bot " + config.Token)
	if err != nil {
		log.Fatal(err)
	}
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Bot is up!")
	})
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.ApplicationCommandData().Name == "clink" {
			link := i.ApplicationCommandData().GetOption("link")
			if link == nil {
				return
			}
			pub := i.ApplicationCommandData().GetOption("public")
			flags := discordgo.MessageFlagsEphemeral
			if pub != nil && pub.BoolValue() {
				flags = 0
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Getting the actual link...",
					Flags:   flags,
				},
			})
			l, err := getLinks(link.StringValue())
			resp := ""
			if err != nil {
				resp = fmt.Sprintf("Error getting %s: %s", link.StringValue(), err)
				log.Println(resp)
			} else {
				resp = l
			}
			_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &resp})
			if err != nil {
				log.Printf("Error sending response: %s", err)
				resp = "Got an error sending reply: " + err.Error()
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &resp})
			}
		}
	})
	tryConnect(dg)
	_, err = dg.ApplicationCommandCreate(config.AppID, "", &discordgo.ApplicationCommand{
		Name:        "clink",
		Description: "Clink the link!",
		Type:        discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "link",
				Description: "link to clink",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "public",
				Description: "show to everyone",
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	dg.ApplicationCommandDelete(config.AppID, "", "clink")
}
