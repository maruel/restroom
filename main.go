// Copyright 2016 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"sort"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/ChimeraCoder/anaconda"
)

type Tweet struct {
	CreatedAt time.Time
	Id        int64
	Place     string
}

type cache struct {
	Users map[string][]Tweet
}

func load() *cache {
	c := &cache{Users: map[string][]Tweet{}}
	f, err := os.Open("restroom.json")
	if err != nil {
		return c
	}
	defer f.Close()
	d := json.NewDecoder(f)
	_ = d.Decode(c)
	if c.Users == nil {
		c.Users = map[string][]Tweet{}
	}
	return c
}

func (c *cache) save() {
	b, err := json.Marshal(c)
	if err != nil {
		log.Fatalf("json: %v", err)
	}
	ioutil.WriteFile("restroom.json", b, 0600)
}

func (c *cache) fetchMore(user, consumerKey, consumerSecret, token, tokenSecret string) error {
	if len(token) == 0 || len(tokenSecret) == 0 {
		return errors.New("both -t and -s are required. If you don't have one, visit https://apps.twitter.com/app/new to create a new token.")
	}
	if len(consumerKey) != 0 {
		anaconda.SetConsumerKey(consumerKey)
	}
	if len(consumerSecret) != 0 {
		anaconda.SetConsumerSecret(consumerSecret)
	}
	api := anaconda.NewTwitterApi(token, tokenSecret)
	defer api.Close()
	// The important bits of
	// https://dev.twitter.com/rest/reference/get/statuses/user_timeline are:
	// - "This method can only return up to 3,200 of a user’s most recent Tweets"
	// - "count" is limited to 200.
	// - Maximum 300 requests / 15 minutes.
	v := url.Values{
		"contributor_details": {"0"},
		"count":               {"200"},
		"exclude_replies":     {"0"},
		"trim_user":           {"1"},
		"include_rts":         {"1"},
		"screen_name":         {user},
	}
	first := true
	ids := map[int64]struct{}{}
	for i := 0; i < 10; i++ {
		if len(c.Users[user]) != 0 {
			// Assumes tweets are in order.
			m := strconv.FormatInt(c.Users[user][len(c.Users[user])-1].Id-1, 10)
			log.Printf("using max_id %s", m)
			v["max_id"] = []string{m}
		}
		log.Printf("Fetching")
		timeline, err := api.GetUserTimeline(v)
		log.Printf("Retrieved %d tweets", len(timeline))
		if err != nil && first {
			return err
		}
		if len(timeline) == 0 || err != nil {
			break
		}
		first = false
		for _, tweet := range timeline {
			if _, ok := ids[tweet.Id]; !ok {
				ids[tweet.Id] = struct{}{}
				t, err := tweet.CreatedAtTime()
				if err != nil {
					log.Fatalf("time: %v", err)
				}
				c.Users[user] = append(c.Users[user], Tweet{t, tweet.Id, tweet.Place.Name})
			}
		}
	}
	return nil
}

func mainImpl() error {
	user := flag.String("u", "", "user to query")
	verbose := flag.Bool("v", false, "verbose output")
	consumerKey := flag.String("k", "", "consumer key")
	consumerSecret := flag.String("c", "", "consumer secret")
	token := flag.String("t", "", "access token")
	tokenSecret := flag.String("s", "", "access token secret")
	flag.Parse()

	if !*verbose {
		log.SetOutput(ioutil.Discard)
	}
	if flag.NArg() != 0 {
		return errors.New("unexpected argument")
	}
	if len(*user) == 0 {
		return errors.New("-u is required")
	}

	c := load()
	defer c.save()
	if len(*token) != 0 {
		if err := c.fetchMore(*user, *consumerKey, *consumerSecret, *token, *tokenSecret); err != nil {
			return err
		}
	}
	hours := [24]int{}
	weekdays := [7]int{}
	placesMap := map[string]int{}
	places := []string{}
	placesLen := 0
	for _, t := range c.Users[*user] {
		//fmt.Printf("%s %s\n", t.CreatedAt.Format("2006-01-02 15:04:05"), t.Place)
		hours[t.CreatedAt.Hour()]++
		weekdays[t.CreatedAt.Weekday()]++
		if len(t.Place) != 0 {
			if _, ok := placesMap[t.Place]; !ok {
				placesMap[t.Place] = 0
				if l := utf8.RuneCountInString(t.Place); l > placesLen {
					placesLen = l
				}
				places = append(places, t.Place)
			}
			placesMap[t.Place]++
		}
	}
	sort.Strings(places)
	fmt.Printf("Processed %d tweets\n", len(c.Users[*user]))
	fmt.Printf("Favorite hour in UTC:\n")
	for i, s := range hours {
		fmt.Printf("  %2d: %3d\n", i, s)
	}
	fmt.Printf("Favorite weekday in UTC:\n")
	for i, s := range weekdays {
		fmt.Printf("  %9s: %3d\n", time.Weekday(i), s)
	}
	fmt.Printf("Favorite places:\n")
	for _, p := range places {
		fmt.Printf("  %*s: %d\n", placesLen, p, placesMap[p])
	}
	return nil
}

func main() {
	if err := mainImpl(); err != nil {
		fmt.Fprintf(os.Stderr, "restroom: %s.\n", err)
		os.Exit(1)
	}
}
