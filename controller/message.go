package controller

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"telepushx/model"
	"time"

	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/time/rate"
)

func PushMessage(c *gin.Context) {

	//user := model.User{ID: c.Param("id")}
	id := c.Param("id")
	users, err := model.GetAllUsers(0, 100000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert id from string to int
	intId, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	activity, err := model.GetActiveContentByID(intId, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting active content"})
		return
	}

	go func() {
		//var keyboard [][]tgbotapi.InlineKeyboardButton
		//for _, button := range activity.Buttons {
		//	keyboardRow := []tgbotapi.InlineKeyboardButton{
		//		tgbotapi.NewInlineKeyboardButtonData(button.Text, button.Link),
		//	}
		//	keyboard = append(keyboard, keyboardRow)
		//}

		//msg := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(os.Getenv("APP_IMAGE_BASE_URL")+"/uploads/"+activity.Image))
		//msg.Caption = activity.Content
		//msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)

		// Create a rate limiter
		limiter := rate.NewLimiter(rate.Limit(30), 1)

		// Create a channel to signal completion
		done := make(chan bool)

		// Start a goroutine to stop the process after maxSendDuration
		go func() {
			time.Sleep(300 * time.Second)
			done <- true
		}()

		bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
		if err != nil {
			log.Panic(err)
		}
		// Send messages concurrently with rate limiting
		var wg sync.WaitGroup
		for _, user := range users {
			wg.Add(1)
			go func(u model.User) {
				defer wg.Done()

				select {
				case <-done:
					log.Printf("Time limit reached. Stopping message sending for user %d", u.ChatId)
					return
				default:
					// Wait for rate limiter
					if err := limiter.Wait(context.Background()); err != nil {
						log.Printf("Rate limiter error for user %d: %v", u.ChatId, err)
						return
					}
					// Send text message with buttons
					//chatID, err := strconv.ParseInt(u.ChatId, 10, 64)
					//if err != nil {
					//	log.Printf("Error parsing ChatID for user %d: %v", u.ChatId, err)
					//	return
					//}
					//msg := tgbotapi.NewMessage(chatID, activity.Content)
					//msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard}
					// Send image message
					if activity.Image != "" {
						chatID, err := strconv.ParseInt(u.ChatId, 10, 64)
						if err != nil {
							log.Printf("Error parsing ChatID for user %d: %v", u.ChatId, err)
							return
						}
						// Parse the JSON array from activity.Image
						var images []string
						err = json.Unmarshal([]byte(activity.Image), &images)
						if err != nil {
							log.Printf("Error parsing image JSON for user %d: %v", u.ChatId, err)
							return
						}

						// Check if there's at least one image
						if len(images) == 0 {
							log.Printf("No images found for user %d", u.ChatId)
							return
						}

						// Get the first image from the array
						firstImage := images[0]
						photo := tgbotapi.NewPhoto(chatID, tgbotapi.FilePath(os.Getenv("APP_IMAGE_BASE_URL")+"/uploads/"+firstImage))
						photo.Caption = activity.Content
						_, err = bot.Send(photo)
						if err != nil {
							log.Printf("Error sending image message to user %d: %v", u.ChatId, err)
						} else {
							log.Printf("Image message sent successfully to user %d", u.ChatId)
						}
					}

					// Wait for rate limiter again
					if err := limiter.Wait(context.Background()); err != nil {
						log.Printf("Rate limiter error for user %d: %v", u.ChatId, err)
						return
					}

				}
			}(*user)
		}

		// Wait for all goroutines to finish or for the time limit to be reached

		// Wait for all goroutines to finish or for the time limit to be reached
		go func() {
			wg.Wait()
			done <- true
		}()
	}()
	//return nil
}
