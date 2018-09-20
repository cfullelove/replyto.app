package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"google.golang.org/appengine" // Required external App Engine library
	"google.golang.org/appengine/datastore"
	aelog "google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

func SendMessage(ctx context.Context, from, to, subject string, headers map[string]string, body []byte) error {

	vals := make(url.Values)
	vals["from"] = []string{from}
	vals["to"] = []string{to}
	vals["subject"] = []string{subject}
	vals["text"] = []string{"Reminder time!\n\n" + string(body)}
	for k, v := range headers {
		vals["h:"+k] = []string{v}
	}

	log.Println(vals.Encode())

	req, err := http.NewRequest("POST", os.Getenv("API_URL")+"/messages", bytes.NewBufferString(vals.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth("api", os.Getenv("API_KEY"))

	resp, err := urlfetch.Client(ctx).Do(req)
	if err != nil {
		return err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	log.Println(bytes.NewBuffer(respBody).String())

	return nil
}

func nextWeekday(start time.Time, weekDay time.Weekday) time.Time {
	t := start.Add(24 * time.Hour)
	for {
		if t.Weekday() == weekDay {
			return t
		}
		t = t.Add(24 * time.Hour)
	}

}

func GetReplytime(from string, tt ...time.Time) (time.Time, error) {
	address := strings.Split(from, "@")
	if len(address) != 2 {
		return time.Now(), fmt.Errorf("invalid address: (%v => %v)", from, address)
	}

	t := time.Now()
	if len(tt) > 0 {
		t = tt[len(tt)-1]
	}

	name := address[0]

	if name == "now" {
		return t, nil
	}

	if res := regexp.MustCompile("^([0-9]+)(d|day|days)$").FindAllStringSubmatch(name, -1); len(res) == 1 {
		if len(res[0]) != 3 {
			return time.Time{}, fmt.Errorf("something went wrong: %v size %d", res[0], len(res[0]))
		}
		days, err := strconv.Atoi(res[0][1])
		return t.Add(time.Duration(days*24) * time.Hour), err
	}

	if res := regexp.MustCompile("^([0-9]+)(h|hour|hours|hrs)$").FindAllStringSubmatch(name, -1); len(res) == 1 {
		if len(res[0]) != 3 {
			return time.Time{}, fmt.Errorf("something went wrong: %v size %d", res[0], len(res[0]))
		}
		hours, err := strconv.Atoi(res[0][1])
		return t.Add(time.Duration(hours) * time.Hour), err
	}

	if res := regexp.MustCompile("(?i)^mon|monday$").FindAllStringSubmatch(name, -1); len(res) == 1 {
		return nextWeekday(t, time.Monday), nil
	}
	if res := regexp.MustCompile("(?i)^tue|tuesday$").FindAllStringSubmatch(name, -1); len(res) == 1 {
		return nextWeekday(t, time.Tuesday), nil
	}
	if res := regexp.MustCompile("(?i)^wed|wednesday$").FindAllStringSubmatch(name, -1); len(res) == 1 {
		return nextWeekday(t, time.Wednesday), nil
	}
	if res := regexp.MustCompile("(?i)^thu|thur|thursday$").FindAllStringSubmatch(name, -1); len(res) == 1 {
		return nextWeekday(t, time.Thursday), nil
	}
	if res := regexp.MustCompile("(?i)^fri|friday$").FindAllStringSubmatch(name, -1); len(res) == 1 {
		return nextWeekday(t, time.Friday), nil
	}
	if res := regexp.MustCompile("(?i)^sat|saturday$").FindAllStringSubmatch(name, -1); len(res) == 1 {
		return nextWeekday(t, time.Saturday), nil
	}
	if res := regexp.MustCompile("(?i)^sun|sunday$").FindAllStringSubmatch(name, -1); len(res) == 1 {
		return nextWeekday(t, time.Sunday), nil
	}

	return time.Now(), fmt.Errorf("couldn't understand name: %v", from)
}

type ReplyMessage struct {
	To                string
	From              string
	Subject           string
	OriginalMessageId string
	Body              []byte
	Schedule          time.Time
}

func main() {

	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Authorization"}
	config.AllowAllOrigins = true
	r.Use(cors.New(config))

	r.POST("/message", func(c *gin.Context) {
		from := c.PostForm("sender")
		toAll := strings.Split(c.PostForm("recipient"), ", ")
		subject := c.PostForm("subject")
		messageId := c.PostForm("Message-Id")
		body := c.PostForm("body-plain")

		buf := &bytes.Buffer{}

		for _, to := range toAll {

			timeToReply, err := GetReplytime(to)
			if err != nil {
				// c.String(http.StatusInternalServerError, "%v", err)
				body = fmt.Sprintf("Warning - error understanding when to reply later!\n\n%v", err)
				timeToReply = time.Now()
			}

			message := ReplyMessage{
				To:                from,
				From:              to,
				Subject:           subject,
				OriginalMessageId: messageId,
				Body:              []byte(body),
				Schedule:          timeToReply,
			}

			ctx := appengine.NewContext(c.Request)

			key, err := datastore.Put(ctx, datastore.NewIncompleteKey(ctx, "message", nil), &message)
			if err != nil {
				c.Error(err)
				return
			}

			fmt.Fprintf(buf, "%v %v", key, message)
			fmt.Fprintln(buf)
		}

		c.String(http.StatusOK, buf.String())

	})

	r.GET("/check_queue", func(c *gin.Context) {
		ctx := appengine.NewContext(c.Request)

		buf := &bytes.Buffer{}

		fmt.Fprintln(buf, "Ready to send:")

		ready := []ReplyMessage{}
		keys, err := datastore.NewQuery("message").Filter("Schedule<=", time.Now()).GetAll(ctx, &ready)
		if err != nil {
			c.Error(err)
			return
		}

		for i, m := range ready {
			fmt.Fprintln(buf, m)

			if err := SendMessage(ctx, m.From, m.To, m.Subject, map[string]string{"In-Reply-To": m.OriginalMessageId}, m.Body); err != nil {
				c.String(http.StatusInternalServerError, "%v", err)
				aelog.Errorf(ctx, err.Error())
				return
			}

			datastore.Delete(ctx, keys[i])

		}

		fmt.Fprintln(buf, "Note yet ready to send:")

		queue := []ReplyMessage{}
		if _, err := datastore.NewQuery("message").Filter("Schedule>=", time.Now()).GetAll(ctx, &queue); err != nil {
			c.Error(err)
			return
		}

		for _, m := range queue {
			fmt.Fprintln(buf, m, m.Schedule.Sub(time.Now()))
		}

		c.String(http.StatusOK, buf.String())

		aelog.Infof(ctx, buf.String())

	})

	http.HandleFunc("/", r.ServeHTTP)

	appengine.Main()

}
