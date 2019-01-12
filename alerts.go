package main

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"text/template"

	"github.com/akerl/go-indefinite-article/indefinite"
	"github.com/akerl/go-linodians/api"
)

var invalidTwitterHandle = strings.Repeat("x", 300)

var messageTemplates map[string][]*template.Template
var messageTemplateStrings = map[string][]string{
	"added": {
		"Congrats to {{.Name}} on their new job as {{.Title}} at @linode, CC @{{.Handle}}",
		"Congrats to {{.Name}} on their new job as {{.Title}}, CC @{{.Handle}}",
		"Congrats to {{.Name}} on their new job as {{.Title}} at @linode",
		"Congrats to {{.Name}} on their new job at @linode, CC @{{.Handle}}",
		"Congrats to {{.Name}} on their new job at @linode",
	},
	"removed": {
		"Best of luck to {{.Name}} in their future endeavors! CC @{{.Handle}}",
		"Best of luck to {{.Name}} in their future endeavors!",
	},
	"modified": {
		"Congrats @linode! {{.Name}} evolved into {{.Title}}! CC @{{.Handle}}",
		"Congrats @linode! {{.Name}} evolved into {{.Title}}!",
	},
}

func init() {
	messageTemplates = make(map[string][]*template.Template)
	for key, stringSlice := range messageTemplateStrings {
		templateSlice := make([]*template.Template, len(stringSlice))
		for index, str := range stringSlice {
			templateSlice[index] = template.Must(template.New("").Parse(str))
		}
		messageTemplates[key] = templateSlice
	}
	log.Print("Loaded templates")
}

type tweetData struct {
	Name   string
	Title  string
	Handle string
}

func alertList(kind string, list api.Company) error {
	log.Printf("Processing %d alerts, kind == %s", len(list), kind)
	for _, e := range list {
		if err := alertIndiv(kind, e); err != nil {
			return err
		}
	}
	return nil
}

func alertIndiv(kind string, e api.Employee) error {
	log.Printf("Processing alert for %s", e.Username)
	options := messageTemplates[kind]
	message, err := buildMessage(options, e)
	if err != nil {
		return err
	}
	log.Printf("Built message: %s", message)
	if kind == "removed" && c.DMRemovals {
		log.Print("Sending via DM")
		return sendDirectMessage(message)
	}
	log.Print("Sending via tweet")
	_, _, err = client.Statuses.Update(message, nil)
	return err
}

func buildMessage(options []*template.Template, e api.Employee) (string, error) {
	aTitle := indefinite.AddArticle(e.Title)
	log.Printf("Parsed indefinite article as: %s", aTitle)
	twitterURL := e.Social["twitter"]
	var twitterHandle string
	if twitterURL == "" {
		twitterHandle = invalidTwitterHandle
	} else {
		twitterParts := strings.Split(twitterURL, "/")
		twitterHandle = twitterParts[len(twitterParts)-1]
	}
	log.Printf("Parsed Twitter handle as %s", twitterHandle)
	td := tweetData{
		Name:   e.Fullname,
		Title:  aTitle,
		Handle: twitterHandle,
	}
	for _, opt := range options {
		var res bytes.Buffer
		err := opt.Execute(&res, td)
		if err != nil {
			return "", err
		}
		if res.Len() <= 140 {
			return res.String(), nil
		}
	}
	return "", fmt.Errorf("no suitable template found")
}
