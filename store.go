package main

import (
	"context"
	"fmt"
	"strconv"
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

func (d *store) GetMessage(key string) (Message, error) {
	var storeKey *datastore.Key
	var message Message

	// The key string might be a numberID, not an encoded string...
	if keyIntID, err := strconv.ParseInt(key, 10, 64); err == nil {
		storeKey = datastore.NewKey(d.ctx, "message_log", "", keyIntID, nil)
	} else {
		storeKey, err = datastore.DecodeKey(key)
		if err != nil {
			return message, err
		}
	}

	if storeKey.Kind() != "message_log" {
		return message, fmt.Errorf("key reference incorrect kind: %v", storeKey.Kind())
	}

	err := datastore.Get(d.ctx, storeKey, &message)
	return message, err

}

func (d *store) GetMessageNumberID(key int64) (Message, error) {
	storeKey := datastore.NewKey(d.ctx, "message_log", "", key, nil)
	return d.GetMessage(storeKey.Encode())
}

type QueryOption func(*datastore.Query) *datastore.Query

func LimitTo(n int) QueryOption {
	return func(q *datastore.Query) *datastore.Query {
		return q.Limit(n)
	}
}

func OrderBy(fieldName string) QueryOption {
	return func(q *datastore.Query) *datastore.Query {
		return q.Order(fieldName)
	}
}

func FilterBy(filterStr string, value interface{}) QueryOption {
	return func(q *datastore.Query) *datastore.Query {
		return q.Filter(filterStr, value)
	}
}

func (d *store) GetMessages(opts ...QueryOption) ([]*datastore.Key, []Message, error) {

	q := datastore.NewQuery("message_log")
	for _, opt := range opts {
		q = opt(q)
	}

	var messages []Message
	keys, err := q.GetAll(d.ctx, &messages)

	return keys, messages, err
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

func (d *store) GetPendingReplies(t time.Time) (map[*datastore.Key]ReplyMessage, error) {
	replies := []ReplyMessage{}

	keys, err := datastore.NewQuery("message").Filter("Schedule>=", t).GetAll(d.ctx, &replies)
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
