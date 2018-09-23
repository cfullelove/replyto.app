package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"google.golang.org/appengine"
	aelog "google.golang.org/appengine/log"
)

func main() {

	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Authorization"}
	config.AllowAllOrigins = true
	r.Use(cors.New(config))

	r.POST("/message", func(c *gin.Context) {
		ctx := appengine.NewContext(c.Request)

		message := NewMessageFromRequest(c.Request)

		aelog.Infof(ctx, "Received message from %v (%v) to %v", message.Sender, message.Subject, message.Recipients)
		replies := message.GetReplyMessages()

		store := NewDataStore(ctx)
		_, err := store.PutReplyMessages(replies)
		if err != nil {
			aelog.Errorf(ctx, "Error storing replies: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		for _, r := range replies {
			aelog.Infof(ctx, "Queued message '%s' to be sent to %s on %s", r.Subject, r.To, r.Schedule.Format(time.ANSIC))
		}

		if _, err := store.PutMessageLog(message); err != nil {
			aelog.Errorf(ctx, "Error storing message log: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.String(http.StatusOK, "%v replies queued", len(replies))
	})

	r.GET("/message/:key", func(c *gin.Context) {
		ctx := appengine.NewContext(c.Request)
		store := NewDataStore(ctx)

		key := c.Param("key")

		message, err := store.GetMessage(key)
		if err != nil {
			aelog.Errorf(ctx, "Error getting message log: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
		}

		c.IndentedJSON(http.StatusOK, message)
	})

	r.GET("/messages", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/messages/5")
	})

	r.GET("/messages/:limit", func(c *gin.Context) {
		ctx := appengine.NewContext(c.Request)
		store := NewDataStore(ctx)

		limit, err := strconv.Atoi(c.Param("limit"))
		if err != nil {
			aelog.Errorf(ctx, "Error getting message log: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		keys, messages, err := store.GetMessages(
			LimitTo(limit),
			OrderBy("-ReceivedTime"),
		)
		if err != nil {
			aelog.Errorf(ctx, "Error getting message log: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		for i := range messages {
			messages[i].Key = keys[i]
		}

		c.IndentedJSON(http.StatusOK, messages)
	})

	r.Any("/check_queue", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/queue/ready")
	})

	r.GET("/queue/ready", func(c *gin.Context) {
		ctx := appengine.NewContext(c.Request)

		store := NewDataStore(ctx)

		replies, err := store.GetReadyReplies(time.Now())
		if err != nil {
			aelog.Errorf(ctx, "Error getting replies: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		for key, m := range replies {
			if appengine.IsDevAppServer() {
				aelog.Infof(ctx, "Would have sent '%v' to %s", m.Subject, m.To)
				store.Delete(key)
				continue
			}

			if err := SendMessage(ctx, m); err != nil {
				aelog.Errorf(ctx, "Error sending reply: %v", err)
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			store.Delete(key)
			aelog.Infof(ctx, "Sent '%v' to %s", m.Subject, m.To)
		}

		c.String(http.StatusOK, "%v replies have been sent", len(replies))

	})

	r.GET("/queue/pending", func(c *gin.Context) {
		ctx := appengine.NewContext(c.Request)

		store := NewDataStore(ctx)

		replies, err := store.GetPendingReplies(time.Now())
		if err != nil {
			aelog.Errorf(ctx, "Error getting replies: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		repliesMap := make(map[time.Time]ReplyMessage)

		for _, m := range replies {
			repliesMap[m.Schedule] = m
		}

		c.IndentedJSON(http.StatusOK, repliesMap)
	})

	http.HandleFunc("/", r.ServeHTTP)

	appengine.Main()

}
