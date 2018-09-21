package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"google.golang.org/appengine" // Required external App Engine library
	"google.golang.org/appengine/datastore"
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

		key := c.Param("key")
		log.Println(key)

		var message Message

		var storeKey *datastore.Key
		if keyIntID, err := strconv.ParseInt(key, 10, 64); err == nil {
			storeKey = datastore.NewKey(ctx, "message_log", "", keyIntID, nil)
		} else {
			aelog.Debugf(ctx, "unable to parse %s as IntID: %v", err)
		}

		if decKey, err := datastore.DecodeKey(key); err == nil {
			storeKey = decKey
		} else {
			aelog.Debugf(ctx, "unable to decode %s as key: %v", err)
		}

		if err := datastore.Get(ctx, storeKey, &message); err != nil {
			aelog.Errorf(ctx, "Error getting message log: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.IndentedJSON(http.StatusOK, message)
	})

	r.GET("/messages", func(c *gin.Context) {
		ctx := appengine.NewContext(c.Request)

		var messages []Message
		keys, err := datastore.NewQuery("message_log").Limit(5).GetAll(ctx, &messages)
		if err != nil {
			aelog.Errorf(ctx, "Error getting message log: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		for _, key := range keys {
			log.Println(key.StringID(), key.IntID(), key.Encode())
			// datastore.Get(ctx, datastore.NewKey(ctx, "message_log", key.StringID()))
		}

		c.IndentedJSON(http.StatusOK, messages)
	})

	r.GET("/check_queue", func(c *gin.Context) {
		ctx := appengine.NewContext(c.Request)

		store := NewDataStore(ctx)

		replies, err := store.GetReadyReplies(time.Now())
		if err != nil {
			aelog.Errorf(ctx, "Error getting replies: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		for key, m := range replies {
			log.Println(m)
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

	http.HandleFunc("/", r.ServeHTTP)

	appengine.Main()

}
