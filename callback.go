package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"google.golang.org/appengine/datastore"
)

type Message struct {
	Sender       string
	Recipients   []string
	Subject      string
	MessageID    string
	ReceivedTime time.Time
	Body         []byte
	Request      struct {
		Headers http.Header
		Body    []byte
	} `datastore:"-"`
	RequestRaw []byte
}

func (m *Message) Load(ps []datastore.Property) error {
	if err := datastore.LoadStruct(m, ps); err != nil {
		return err
	}

	if err := json.Unmarshal(m.RequestRaw, &(m.Request)); err != nil {
		return err
	}

	return nil
}

func (m *Message) Save() ([]datastore.Property, error) {
	b, err := json.MarshalIndent(m.Request, "", "    ")
	if err != nil {
		return nil, err
	}
	m.RequestRaw = b

	return datastore.SaveStruct(m)
}

func NewMessageFromRequest(req *http.Request) Message {

	var message Message

	message.Sender = req.PostFormValue("sender")
	message.Recipients = strings.Split(req.PostFormValue("recipient"), ", ")
	message.Subject = req.PostFormValue("subject")
	message.MessageID = req.PostFormValue("Message-Id")
	message.Body = []byte(req.PostFormValue("body-plain"))
	message.Request.Headers = req.Header
	// body, err := req.GetBody()
	// if err == nil {
	// message.Request.Body, _ = ioutil.ReadAll(body)
	// }
	message.Request.Body, _ = ioutil.ReadAll(req.Body)

	return message
}

func (m Message) RecievedTime() time.Time {
	return time.Now()
}

func (m Message) newReplyMessage(to string, t time.Time) ReplyMessage {
	return ReplyMessage{
		To:                m.Sender,
		From:              to,
		Subject:           m.Subject,
		OriginalMessageId: m.MessageID,
		Headers: map[string]string{
			"In-Reply-To": m.MessageID,
		},
		Body:     m.Body,
		Schedule: t,
	}
}

func (m Message) GetReplyMessages() []ReplyMessage {
	replies := []ReplyMessage{}

	for _, to := range m.Recipients {
		reply := m.newReplyMessage(to, m.RecievedTime())

		timeToReply, err := getReplytime(to, m.RecievedTime())
		if err != nil {
			str := "Error: there was an error determing when we should reply to your email.\n\nThe address you used was %s and it's either not supported of we had a problem (%v)"
			reply.Body = []byte(fmt.Sprintf(str, to, err))
		} else {
			reply.Schedule = timeToReply
			reply.Body = append([]byte(fmt.Sprintf("Replying to your message originally sent %v to %v\n\n\n", m.RecievedTime().Format(time.ANSIC), to)), reply.Body...)
		}

		replies = append(replies, reply)
	}

	return replies
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

func getReplytime(from string, t time.Time) (time.Time, error) {
	address := strings.Split(from, "@")
	if len(address) != 2 {
		return time.Now(), fmt.Errorf("invalid address: (%v => %v)", from, address)
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
