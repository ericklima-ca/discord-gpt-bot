package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	_ "github.com/joho/godotenv/autoload"
	"github.com/sashabaranov/go-openai"

	"github.com/bwmarrin/discordgo"
)

var contents []openai.ChatCompletionMessage

var client = openai.NewClient(os.Getenv("OPENAI_API_KEY"))

func main() {
	discordAppToken := os.Getenv("DISCORD_APP_TOKEN")
	dg, err := discordgo.New("Bot " + discordAppToken)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	dg.AddHandler(func(s *discordgo.Session, m *discordgo.Ready) {
		fmt.Println("Bot running!")
	})
	dg.AddHandler(messageCreate)

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	err = dg.Open()
	defer dg.Close()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	fmt.Println("Closing bot...")
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if strings.HasPrefix(m.Content, "/reset") {
		contents = []openai.ChatCompletionMessage{}
		_, err := s.ChannelMessageSend(os.Getenv("CHANNEL_ID"), "Resetando memória...")
		if err != nil {
			log.Fatal(err)
			return
		}
		s.ChannelMessageSend(os.Getenv("CHANNEL_ID"), "Memória resetada...")
		return
	}

	// if strings.HasPrefix(m.Content, "/dalle") {
	// 	msg := sendImage(strings.ReplaceAll(m.Content, "/dalle", ""))
	// 	s.ChannelMessageSendComplex(os.Getenv("CHANNEL_ID"), msg)
	// 	return
	// }

	if !strings.HasPrefix(m.Content, "/chat") {
		return
	}

	content := strings.ReplaceAll(m.Content, "/chat ", "")

	if len(contents) > 5 {
		contents = contents[0:5]
	}
	contents = append(contents, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: content,
	})
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: append([]openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a Discord Bot named as Jarvinho who talks about everything in portuguese. Your answers MUST be a maximum of 200 characters, so BE BRIEF and to the point.",
				},
			}, contents...),
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return
	}
	contents = append(contents,
		openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: resp.Choices[0].Message.Content,
		},
	)
	_, err = s.ChannelMessageSend(os.Getenv("CHANNEL_ID"), resp.Choices[0].Message.Content)
	if err != nil {
		fmt.Println("error sending DM message:", err)
		s.ChannelMessageSend(
			os.Getenv("CHANNEL_ID"),
			"Failed to send you a DM. "+
				"Did you disable DM in your privacy settings?",
		)
	}
}
func sendImage(description string) *discordgo.MessageSend {
	ctx := context.Background()
	reqBase64 := openai.ImageRequest{
		Prompt:         description,
		Size:           openai.CreateImageSize1024x1024,
		ResponseFormat: openai.CreateImageResponseFormatB64JSON,
		N:              1,
	}

	respBase64, err := client.CreateImage(ctx, reqBase64)
	if err != nil {
		fmt.Printf("Image creation error: %v\n", err)
		return nil
	}

	imgBytes, err := base64.StdEncoding.DecodeString(respBase64.Data[0].B64JSON)
	if err != nil {
		fmt.Printf("Base64 decode error: %v\n", err)
		return nil
	}

	r := bytes.NewReader(imgBytes)

	msg := &discordgo.MessageSend{
		File: &discordgo.File{
			Name:        "dalle.png",
			ContentType: "image/png",
			Reader:      r,
		},
	}
	return msg
}
