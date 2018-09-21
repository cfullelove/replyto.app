package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	aelog "google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

// func SendMessage(ctx context.Context, from, to, subject string, headers map[string]string, body []byte) error {
func SendMessage(ctx context.Context, m ReplyMessage) error {

	vals := make(url.Values)
	vals.Add("from", m.From)
	vals.Add("to", m.To)
	vals.Add("subject", m.Subject)
	vals.Add("text", string(m.Body))
	for k, v := range m.Headers {
		vals.Add("h:"+k, v)
	}

	aelog.Infof(ctx, vals.Encode())

	cfg := Config()
	req, err := http.NewRequest("POST", cfg.Get("api_url")+"/messages", bytes.NewBufferString(vals.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth("api", cfg.Get("api_key"))

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
