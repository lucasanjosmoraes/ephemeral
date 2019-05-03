package main

import (
	"encoding/json"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"fmt"

	"github.com/ChimeraCoder/anaconda"
)

var (
	consumerKey       = getenv("TWITTER_CONSUMER_KEY")
	consumerSecret    = getenv("TWITTER_CONSUMER_SECRET")
	accessToken       = getenv("TWITTER_ACCESS_TOKEN")
	accessTokenSecret = getenv("TWITTER_ACCESS_TOKEN_SECRET")
	maxTweetAge       = getenv("MAX_TWEET_AGE")
	whitelist         = getWhitelist()
)

func getenv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		panic("missing required environment variable " + name)
	}
	return v
}

func getWhitelist() []string {
	v := os.Getenv("WHITELIST")

	if v == "" {
		return make([]string, 0)
	}

	return strings.Split(v, ":")
}

func getTimeline(api *anaconda.TwitterApi, sinceID *int64) ([]anaconda.Tweet, error) {
	args := url.Values{}
	args.Add("count", "200") // Twitter only returns most recent 20 tweets by default, so override
	if sinceID != nil {
		fmt.Println("Getting timeline since ID:", strconv.FormatInt(*sinceID, 10))
		args.Add("max_id", strconv.FormatInt(*sinceID, 10))
	}
	args.Add("include_rts", "true") // When using count argument, RTs are excluded, so include them as recommended
	timeline, err := api.GetUserTimeline(args)
	if err != nil {
		return make([]anaconda.Tweet, 0), err
	}
	return timeline, nil
}

func isWhitelisted(id int64) bool {
	tweetID := strconv.FormatInt(id, 10)

	for _, w := range whitelist {
		if w == tweetID {
			return true
		}
	}
	return false
}

func deleteTweets(tweets []anaconda.Tweet, api *anaconda.TwitterApi, ageLimit time.Duration) (int64, error) {
	fmt.Println("Checking if need to delete tweets")

	var lastID int64
	var lastTime time.Duration

	for _, t := range tweets {
		createdTime, err := t.CreatedAtTime()
		if err != nil {
			fmt.Println("could not parse time ", err)
			return 0, err
		} else {
			if time.Since(createdTime) > ageLimit && !isWhitelisted(t.Id) {
				_, err := api.DeleteTweet(t.Id, true)
				fmt.Println("DELETED ID ", t.Id)
				fmt.Println("TWEET ", createdTime, " - ", t.Text)
				if err != nil {
					fmt.Println("failed to delete: ", err)
					return 0, err
				}
			}
		}
		if time.Since(createdTime) > lastTime {
			lastTime = time.Since(createdTime)
			lastID = t.Id
		}
	}
	fmt.Println("Finish this iteration and sleeping for some seconds")
	time.Sleep(3 * time.Second)
	return lastID, nil
}

func deleteFromTimeline(api *anaconda.TwitterApi, ageLimit time.Duration) {
	fmt.Println("Calling timeline")
	timeline, _ := getTimeline(api, nil)
	fmt.Println("Timeline retrieved, length: ", len(timeline))

	for len(timeline) > 1 {
		lastID, err := deleteTweets(timeline, api, ageLimit)

		if err == nil {
			fmt.Println("Calling timeline")
			timeline, err = getTimeline(api, &lastID)
			fmt.Println("Timeline retrieved, length: ", len(timeline))
		}
	}

	fmt.Println("no more tweets to delete")
}

func ephemeral() error {
	anaconda.SetConsumerKey(consumerKey)
	anaconda.SetConsumerSecret(consumerSecret)
	api := anaconda.NewTwitterApi(accessToken, accessTokenSecret)
	api.SetLogger(anaconda.BasicLogger)

	h, _ := time.ParseDuration(maxTweetAge)

	deleteFromTimeline(api, h)

	return nil
}

func main() {
	err := ephemeral()

	if err != nil {
		fmt.Println(err)
	}
}

type Result struct {
	Tweets []string `json: "tweets"`
}

// Delete all tweets in a JSON file downloaded from Twitter
func deleteFromJSON() {
	plan, _ := ioutil.ReadFile("/Users/LucasLTMD/Downloads/twitter-2019-05-02-c9766e896742b572b2b4fe4c9b3d6735ca4727a5ced2cde659f8f4f9bd45ca1d/oldTweets.json")
	timeline := Result{}
	err := json.Unmarshal(plan, &timeline)
	if err != nil {
		fmt.Println("Error unmarshal this shit", err)
	}

	anaconda.SetConsumerKey(consumerKey)
	anaconda.SetConsumerSecret(consumerSecret)
	api := anaconda.NewTwitterApi(accessToken, accessTokenSecret)
	api.SetLogger(anaconda.BasicLogger)

	for i := 0; i <= len(timeline.Tweets); i++ {
		if i%200 == 0 {
			fmt.Println("sleeping")
			time.Sleep(5 * time.Second)
		}
		tweetID, err := strconv.ParseInt(timeline.Tweets[i], 10, 64)
		if err != nil {
			continue
		}
		_, err = api.DeleteTweet(tweetID, true)
		fmt.Println("DELETED ID ", tweetID)
		if err != nil {
			fmt.Println("failed to delete: ", err)
			continue
		}
	}
}
