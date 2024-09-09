package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"telepushx/model"

	_ "github.com/go-sql-driver/mysql" // Assuming MySQL, adjust if using a different database
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/time/rate"
)

const (
	telegramAPIBaseURL = "https://api.telegram.org/bot"
	botToken           = "6253545273:AAEzdFppjluWM_QplxoZiMsi0GJzDgNmEVI"  // Replace with your actual bot token
	dbConnectionString = "user:password@tcp(localhost:3306)/database_name" // Replace with your actual database connection string
	maxSendRate        = 30                                                // Maximum messages per second
	maxSendDuration    = 20 * time.Minute                                  // Maximum duration for sending messages
)

func main() {
	// Connect to the database
	db, err := sql.Open("mysql", dbConnectionString)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// Initialize bot
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("Error creating bot: %v", err)
	}
	fmt.Println("Bot created successfully!\n", bot.Token)

	// Get users and active content
	users, err := model.GetAllUsers(db)
	if err != nil {
		log.Fatalf("Error getting users: %v", err)
	}

	activeContent, err := model.GetActiveContentByID(db, 1)
	if err != nil {
		log.Fatalf("Error getting active content: %v", err)
	}

	// Create keyboard from buttons
	var keyboard [][]tgbotapi.InlineKeyboardButton
	for _, button := range activeContent.Buttons {
		keyboardRow := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(button.Text, button.Link),
		}
		keyboard = append(keyboard, keyboardRow)
	}

	// Create a rate limiter
	limiter := rate.NewLimiter(rate.Limit(maxSendRate), 1)

	// Create a channel to signal completion
	done := make(chan bool)

	// Start a goroutine to stop the process after maxSendDuration
	go func() {
		time.Sleep(maxSendDuration)
		done <- true
	}()

	// Send messages concurrently with rate limiting
	var wg sync.WaitGroup
	for _, user := range users {
		wg.Add(1)
		go func(u model.User) {
			defer wg.Done()

			select {
			case <-done:
				log.Printf("Time limit reached. Stopping message sending for user %d", u.ID)
				return
			default:
				// Wait for rate limiter
				if err := limiter.Wait(context.Background()); err != nil {
					log.Printf("Rate limiter error for user %d: %v", u.ID, err)
					return
				}
				// Send text message with buttons
				chatID, err := strconv.ParseInt(u.ChatID, 10, 64)
				if err != nil {
					log.Printf("Error parsing ChatID for user %d: %v", u.ID, err)
					return
				}
				msg := tgbotapi.NewMessage(chatID, activeContent.Content)
				msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard}
				// Send image message
				if activeContent.Image != "" {
					chatID, err := strconv.ParseInt(u.ChatID, 10, 64)
					if err != nil {
						log.Printf("Error parsing ChatID for user %d: %v", u.ID, err)
						return
					}
					photo := tgbotapi.NewPhoto(chatID, tgbotapi.FilePath(activeContent.Image))
					photo.Caption = activeContent.Title
					_, err = bot.Send(photo)
					if err != nil {
						log.Printf("Error sending image message to user %d: %v", u.ID, err)
					} else {
						fmt.Printf("Image message sent successfully to user %d!\n", u.ID)
					}
				}
				_, err = bot.Send(msg)
				if err != nil {
					log.Printf("Error sending text message with buttons to user %d: %v", u.ID, err)
				} else {
					fmt.Printf("Text message with buttons sent successfully to user %d!\n", u.ID)
				}

				// Wait for rate limiter again
				if err := limiter.Wait(context.Background()); err != nil {
					log.Printf("Rate limiter error for user %d: %v", u.ID, err)
					return
				}

			}
		}(user)
	}

	// Wait for all goroutines to finish or for the time limit to be reached
	go func() {
		wg.Wait()
		done <- true
	}()

	<-done
	fmt.Println("Message sending process completed or timed out!")
}
