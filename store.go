package main

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/appengine/datastore"
)

type ReplyMessage struct {
	To                string
	From              string
	Subject           string
	OriginalMessageId string
	Headers           map[string]string `datastore:"-"`
	HeaderKeys        []string
	HeaderVals        []string
	Body              []byte
	Schedule          time.Time
}

func (rm *ReplyMessage) Load(ps []datastore.Property) error {
	if err := datastore.LoadStruct(rm, ps); err != nil {
		return err
	}

	rm.Headers = make(map[string]string)
	for i, k := range rm.HeaderKeys {
		if i >= len(rm.HeaderVals) {
			break
		}
		rm.Headers[k] = rm.HeaderVals[i]
	}

	return nil
}

func (rm *ReplyMessage) Save() ([]datastore.Property, error) {
	rm.HeaderKeys = []string{}
	rm.HeaderVals = []string{}
	for k, v := range rm.Headers {
		rm.HeaderKeys = append(rm.HeaderKeys, k)
		rm.HeaderVals = append(rm.HeaderVals, v)
	}
	return datastore.SaveStruct(rm)
}

type store struct {
	ctx context.Context
}

func NewDataStore(ctx context.Context) *store {
	return &store{
		ctx: ctx,
	}
}

func (d *store) PutMessageLog(message Message) (*datastore.Key, error) {
	key, err := datastore.Put(d.ctx, datastore.NewIncompleteKey(d.ctx, "message_log", nil), &message)
	if err != nil {
		return nil, fmt.Errorf("error storing message: %v", err)
	}

	return key, err
}

func (d *store) PutReplyMessage(message ReplyMessage) (*datastore.Key, error) {
	key, err := datastore.Put(d.ctx, datastore.NewIncompleteKey(d.ctx, "message", nil), &message)
	if err != nil {
		return nil, fmt.Errorf("error storing reply message: %v", err)
	}

	return key, err
}

func (d *store) PutReplyMessages(messages []ReplyMessage) ([]*datastore.Key, error) {

	keys := []*datastore.Key{}
	for i := 0; i < len(messages); i++ {
		keys = append(keys, datastore.NewIncompleteKey(d.ctx, "message", nil))
	}

	keys, err := datastore.PutMulti(d.ctx, keys, messages)
	if err != nil {
		return nil, fmt.Errorf("error storing reply message: %v", err)
	}

	return keys, err
}

func (d *store) GetReadyReplies(t time.Time) (map[*datastore.Key]ReplyMessage, error) {
	replies := []ReplyMessage{}

	keys, err := datastore.NewQuery("message").Filter("Schedule<=", t).GetAll(d.ctx, &replies)
	if err != nil {
		return nil, fmt.Errorf("couldn't get reply messages: %v", err)
	}

	repliesMap := make(map[*datastore.Key]ReplyMessage)
	for i, r := range replies {
		repliesMap[keys[i]] = r
	}

	return repliesMap, nil
}

func (d *store) Delete(key *datastore.Key) error {
	return datastore.Delete(d.ctx, key)
}
