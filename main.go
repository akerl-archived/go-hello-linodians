package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/akerl/go-lambda/s3"
	"github.com/aws/aws-lambda-go/lambda"
)

type config struct {
	ConsumerKey    string `json:"consumer_key"`
	ConsumerSecret string `json:"consumer_secret"`
	AccessToken    string `json:"access_token"`
	AccessSecret   string `json:"access_secret"`
	DMRemovals     bool   `json:"dm_removals"`
	DMTarget       string `json:"dm_target"`
	CacheFile      string `json:"cache_file"`
}

var c config

func handler() error {
	return nil
}

func loadConfig() {
	cf, err := s3.GetConfigFromEnv(&c)
	if err != nil {
		panic(err)
	}
	cf.OnError = func(_ *s3.ConfigFile, err error) {
		log.Print(err)
	}
	cf.Autoreload(60)
}

func loadClient() {
}

func main() {
	loadConfig()
	loadClient()
	lambda.Start(handler)
}
