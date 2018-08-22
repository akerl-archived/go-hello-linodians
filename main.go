package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/akerl/go-lambda/s3"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/awserr"
	s3api "github.com/aws/aws-sdk-go/service/s3"
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
var client twitter.Client
var bucket string

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
	config := oauth1.NewConfig(c.ConsumerKey, c.ConsumerSecret)
	token := oauth1.NewToken(c.AccessToken, c.AccessSecret)
	httpClient := config.Client(oauth1.NoContext, token)

	client = twitter.NewClient(httpClient)

	verifyParams := &twitter.AccountVerifyParams{
		SkipStatus:   twitter.Bool(true),
		IncludeEmail: twitter.Bool(true),
	}
	_, _, err := client.Accounts.VerifyCredentials(verifyParams)
	if err != nil {
		panic(err)
	}
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

	err = sanityCheck(newList)
	if err != nil {
		return err
	}

	oldList, err := loadOld(newList)
	if err != nil {
		return err
	}

	ds := api.Diff(oldList, newList)

	err = alert("added", ds.Added)
	if err != nil {
		return err
	}
	err = alert("modified", ds.Modified)
	if err != nil {
		return err
	}
	err = alert("removed", ds.Removed)
	if err != nil {
		return err
	}

	return writeOld(oldList)
}

var messages = map[string][]string{
	"added": []string{
		"Congrats to %<fullname>s on their new job as %<a_title>s at @linode, CC @%<twitter>s",
		"Congrats to %<fullname>s on their new job as %<a_title>s at @linode",
		"Congrats to %<fullname>s on their new job at @linode, CC @%<twitter>s",
		"Congrats to %<fullname>s on their new job at @linode",
	},
	"removed": []string{
		"Best of luck to %<fullname>s in their future endeavors! CC @%<twitter>s",
		"Best of luck to %<fullname>s in their future endeavors!",
	},
	"modified": []string{
		"Congrats @linode! %<fullname>s evolved into %<a_title>s! CC @%<twitter>s",
		"Congrats @linode! %<fullname>s evolved into %<a_title>s!",
	},
}

func alert(kind string, list api.Company) error {
	// Build message here
	if kind == "removed" && c.DMRemoval {
		// Do via DM
	} else {
		// Do via Tweet
	}
}

func loadOld(new api.Company) (api.Company, error) {
	obj, err := s3.GetObject(bucket, c.CacheFile)
	if err != nil {
		aerr, ok := err.(awserr.Error)
		if ok && aerr.Code() == s3api.ErrCodeNoSuchKey {
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
	input := &s3api.PutObjectInput{
		Body:   bytes.NewReader(data),
		Bucket: bucket,
		Key:    c.CacheFile,
	}
	s3client := s3.Client()
	_, err := s3client.PutObject(input)
	return err
}

func sanityCheck(new api.Company) error {
	if _, ok := new["caker"]; !ok {
		return fmt.Errorf("caker is missing; assuming results invalid")
	}
	return nil
}
