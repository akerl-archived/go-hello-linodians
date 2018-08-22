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
		"Congrats to {{.Name}} on their new job as {{.Title}} at @linode, CC @{{.Handle}}",
		"Congrats to {{.Name}} on their new job as {{.Title}}, CC @{{.Handle}}",
		"Congrats to {{.Name}} on their new job as {{.Title}} at @linode",
		"Congrats to {{.Name}} on their new job at @linode, CC @{{.Handle}}",
		"Congrats to {{.Name}} on their new job at @linode",
	},
	"removed": []string{
		"Best of luck to {{.Name}} in their future endeavors! CC @{{.Handle}}",
		"Best of luck to {{.Name}} in their future endeavors!",
	},
	"modified": []string{
		"Congrats @linode! {{.Name}} evolved into {{.Title}}! CC @{{.Handle}}",
		"Congrats @linode! {{.Name}} evolved into {{.Title}}!",
	},
}

type tweetData struct {
	Name   string
	Title  string
	Handle string
}

// Sourced from https://github.com/rossmeissl/indefinite_article
//  A_REQUIRING_PATTERNS = /^(([bcdgjkpqtuvwyz]|onc?e|onearmed|onetime|ouija)$|e[uw]|uk|ubi|ubo|oaxaca|ufo|ur[aeiou]|use|ut([^t])|unani|uni(l[^l]|[a-ko-z]))/i
//  AN_REQUIRING_PATTERNS = /^([aefhilmnorsx]$|hono|honest|hour|heir|[aeiou]|8|11)/i
//  UPCASE_A_REQUIRING_PATTERNS = /^(UN$)/
//  UPCASE_AN_REQUIRING_PATTERNS = /^$/ #need if we decide to support acronyms like "XL" (extra-large)

//  def indefinite_article
//    first_word = to_s.split(/[- ]/).first
//    if (first_word[AN_REQUIRING_PATTERNS] || first_word[UPCASE_AN_REQUIRING_PATTERNS]) &&
//       !(first_word[A_REQUIRING_PATTERNS] || first_word[UPCASE_A_REQUIRING_PATTERNS])
//      'an'
//    else
//      'a'
//    end unless first_word.nil?
//  end

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
