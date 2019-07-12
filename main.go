package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/akerl/go-lambda/s3"
	"github.com/akerl/go-linodians/api"
	"github.com/aws/aws-lambda-go/lambda"
	s3api "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/dghubble/sling"
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
var dmClient *sling.Sling
var client *twitter.Client
var bucket string

func loadConfig() {
	cf, err := s3.GetConfigFromEnv(&c)
	if err != nil {
		panic(err)
	}
	log.Print("Loaded config")
	cf.OnError = func(_ *s3.ConfigFile, err error) {
		log.Printf("Error reloading config: %s", err)
	}
	cf.Autoreload(60)
}

func loadClient() {
	config := oauth1.NewConfig(c.ConsumerKey, c.ConsumerSecret)
	token := oauth1.NewToken(c.AccessToken, c.AccessSecret)
	httpClient := config.Client(oauth1.NoContext, token)

	dmClient = sling.New().Client(httpClient).Base("https://api.twitter.com/1.1/direct_messages/")
	client = twitter.NewClient(httpClient)
	log.Print("Client configured")

	verifyParams := &twitter.AccountVerifyParams{
		SkipStatus:   twitter.Bool(true),
		IncludeEmail: twitter.Bool(true),
	}
	_, _, err := client.Accounts.VerifyCredentials(verifyParams)
	if err != nil {
		panic(err)
	}
	log.Print("Client validated")
}

func main() {
	loadConfig()
	loadClient()
	bucket = os.Getenv("S3_BUCKET")
	lambda.Start(handler)
}

func handler() error {
	newList, err := api.Load()
	if err != nil {
		return err
	}
	log.Printf("Loaded new list with %d entries", len(newList))

	err = sanityCheck(newList)
	if err != nil {
		return err
	}
	log.Print("List passed sanity checks")

	oldList, err := loadOld(newList)
	if err != nil {
		return err
	}
	log.Printf("Loaded old list with %d entries", len(oldList))

	ds := api.Diff(oldList, newList)
	log.Printf(
		"Loaded diff: added %d, modified %d, removed %d",
		len(ds.Added),
		len(ds.Modified),
		len(ds.Removed),
	)

	err = alertList("added", ds.Added)
	if err != nil {
		return err
	}
	err = alertList("modified", ds.Modified)
	if err != nil {
		return err
	}
	err = alertList("removed", ds.Removed)
	if err != nil {
		return err
	}

	err = writeOld(newList)
	if err != nil {
		return err
	}
	log.Print("Wrote list to cache")
	return nil
}

func loadOld(new api.Company) (api.Company, error) {
	obj, err := s3.GetObject(bucket, c.CacheFile)
	if err != nil {
		aerr, ok := err.(awserr.Error)
		if ok && aerr.Code() == s3api.ErrCodeNoSuchKey {
			log.Print("cache not found, using new list as old")
			return new, nil
		}
		return api.Company{}, err
	}
	var old api.Company
	err = json.Unmarshal(obj, &old)
	return old, err
}

func writeOld(old api.Company) error {
	data, err := json.MarshalIndent(old, "", "  ")
	if err != nil {
		return err
	}
	s3client, err := s3.Client()
	if err != nil {
		return err
	}
	input := &s3api.PutObjectInput{
		Body:   bytes.NewReader(data),
		Bucket: &bucket,
		Key:    &c.CacheFile,
	}
	req := s3client.PutObjectRequest(input)
	_, err = req.Send(context.Background())
	return err
}

func sanityCheck(new api.Company) error {
	if _, ok := new["caker"]; !ok {
		return fmt.Errorf("caker is missing; assuming results invalid")
	}
	return nil
}
