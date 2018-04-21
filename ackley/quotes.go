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
	quote_found := false
	quote_author_found := false
	for {
		token_type := t.Next()
		switch {
		case token_type == html.ErrorToken:
			glog.Infof("Error: Got an Error Token:%v\n", t.Err())
			return
		case token_type == html.StartTagToken:
			token := t.Token()
			switch {
			case token.Data == "a":
				for _, tokenAttr := range token.Attr {
					//fmt.Printf("Key: %v, Value:%v\n", tokenAttr.Key, tokenAttr.Val)
					if tokenAttr.Key == "class" && strings.Contains(tokenAttr.Val, "b-qt") {
						quote_found = true
					} else if tokenAttr.Key == "class" && strings.Contains(tokenAttr.Val, "bq-aut") {
						quote_author_found = true
					}
				}
			}
		case token_type == html.TextToken:
			if quote_found {
				current_quote = string(t.Text())
				quote_found = false
			}

			if quote_author_found {
				current_quote += " -- " + string(t.Text())
				quote_author_found = false
				glog.Infof("Appending quote to quotes: (%v)\n", current_quote)
				iqr := InspirationalQuotesRequest{Type: "ADD", Val: current_quote}
				inspirational_quotes_req_channel <- iqr
				<-inspirational_quotes_resp_channel
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
