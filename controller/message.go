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
	log.Printf("activity: %v", activity.Content)

	go doPushMessage(activity)

	c.JSON(http.StatusOK, gin.H{"message": "Push process started"})
	//return nil
}

func doPushMessage(activity *model.Activity) {

	users, err := model.GetAllUsers(0, 100000)
	if err != nil {
		log.Printf("Error getting users: %v", err)
		return
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if err != nil {
		log.Printf("Error creating bot: %v", err)
		return
	}

	limiter := rate.NewLimiter(rate.Limit(30), 1)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	var wg sync.WaitGroup
	for _, user := range users {
		wg.Add(1)
		go func(u *model.User) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
				if err := limiter.Wait(ctx); err != nil {
					log.Printf("Rate limit error for user %s: %v", u.ChatId, err)
					return
				}

				chatID, err := strconv.ParseInt(u.ChatId, 10, 64)
				if err != nil {
					log.Printf("Error parsing ChatID for user %s: %v", u.ChatId, err)
					return
				}

				var images []string
				err = json.Unmarshal([]byte(activity.Image), &images)
				if err != nil {
					log.Printf("Error parsing image JSON for user %s: %v", u.ChatId, err)
					return
				}

				if len(images) > 0 {
					photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(os.Getenv("APP_IMAGE_BASE_URL")+"/uploads/"+images[0]))
					photo.Caption = activity.Content
					_, err = bot.Send(photo)
				} else {
					msg := tgbotapi.NewMessage(chatID, activity.Content)
					_, err = bot.Send(msg)
				}

				if err != nil {
					log.Printf("Error sending message to user %s: %v", u.ChatId, err)
				} else {
					log.Printf("Message sent successfully to user %s", u.ChatId)
				}
			}
		}(user)
	}

	wg.Wait()
	log.Println("Push process completed or timed out")
}
