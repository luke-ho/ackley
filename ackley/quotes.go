// Copyright 2016 Luke Ho All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"ackley"
	"bytes"
	"fmt"
	"github.com/golang/glog"
	"golang.org/x/net/html"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

var (
	inspirational_quotes              []string
	inspirational_quotes_req_channel  chan InspirationalQuotesRequest
	inspirational_quotes_resp_channel chan InspirationalQuotesResponse
)

type InspirationalQuotesRequest struct {
	Type string // "GET", "ADD"
	Val  string
}

type InspirationalQuotesResponse struct {
	Type string // "OK", "FAIL"
	Val  string
}

func QuotesInit(ack_init *ackley.AckleyInit) {
	inspirational_quotes = make([]string, 0)
	inspirational_quotes_req_channel = make(chan InspirationalQuotesRequest, 100)
	inspirational_quotes_resp_channel = make(chan InspirationalQuotesResponse, 100)

	go ProcessInspirationalQuote()

	if ack_init.Test_mode == false {
		go PopulateQuotes()
	}
}

func AddQuotes(url string) {
	resp, err := http.Get(url)
	if err != nil {
		glog.Errorf("Unable to download inspirational page:%v\n", err.Error())
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		glog.Errorf("Error while reading in entire body:%v\n", err.Error())
		return
	}

	reader := bytes.NewReader([]byte(string(body)))

	t := html.NewTokenizer(reader)
	current_quote := ""
	ignore := 0

	token_depth := 0
	for {
		token_type := t.Next()
		switch {
		case token_type == html.ErrorToken:
			glog.Infof("Error: Got an Error Token:%v\n", t.Err())
			return
		case token_type == html.EndTagToken:
			token := t.Token()
			if token.Data == "div" {
				if token_depth > 0 {
					token_depth = token_depth - 1
					if token_depth <= 0 {
						current_quote_split := strings.Split(current_quote, "\n")
						current_quote = ""
						line_count := 0
						// Grab first two lines from text
						for _, line := range current_quote_split {
							if line_count >= 2 {
								break
							}
							if len(line) > 0 {
								current_quote += line + "\n"
								line_count++
							}
						}

						// Reformat current quote
						current_quote_split = strings.Split(current_quote, "\n")
						if len(current_quote_split) == 3 {
							current_quote = current_quote_split[0] + " -- " + current_quote_split[1]
						}

						glog.Infof("Appending quote to quotes: (%v)\n", current_quote)

						iqr := InspirationalQuotesRequest{Type: "ADD", Val: current_quote}
						inspirational_quotes_req_channel <- iqr
						<-inspirational_quotes_resp_channel

						current_quote = ""
						token_depth = 0
						// Reset ignore on each current quote append
						ignore = 0
					}
				}
			}
		case token_type == html.StartTagToken:
			token := t.Token()
			switch {
			case token.Data == "div":
				for _, tokenAttr := range token.Attr {
					//if tokenAttr.Key == "class" && tokenAttr.Val == "tlod masonry-brick" {
					if tokenAttr.Key == "class" && tokenAttr.Val == "m-brick grid-item boxy bqQt" {
						glog.Infof("Token depth:%v\n", token_depth)
						token_depth = token_depth + 1
					}
				}
			}

		case token_type == html.TextToken:
			if ignore == 0 && token_depth > 0 {
				text := string(t.Text())
				if len(text) > 0 {
					current_quote += text
				}
			}
		}
	}
}

func ProcessInspirationalQuote() {
	for {
		select {
		case v := <-inspirational_quotes_req_channel:
			switch {
			case v.Type == "GET":
				response := InspirationalQuotesResponse{Type: "OK", Val: ""}
				if len(inspirational_quotes) > 0 {
					rloc := rand.Intn(len(inspirational_quotes))
					response.Val = fmt.Sprintf("%v", inspirational_quotes[rloc])
				} else {
					response.Type = "FAIL"
					response.Val = "Length of inspirational quotes is 0"
				}
				inspirational_quotes_resp_channel <- response
			case v.Type == "ADD":
				response := InspirationalQuotesResponse{Type: "OK", Val: ""}
				inspirational_quotes = append(inspirational_quotes, v.Val)
				inspirational_quotes_resp_channel <- response
			}
		}
	}
}

func AddQuoteFromPage(url string) {
	time.Sleep(time.Second + (time.Duration(rand.Intn(500)) * time.Millisecond))
	AddQuotes(url)
}

func PopulateQuotes() {
	// Thanks brainy quote!
	AddQuoteFromPage("http://www.brainyquote.com/quotes/topics/topic_inspirational.html")
}
