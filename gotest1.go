//http://schier.co/blog/2015/04/26/a-simple-web-scraper-in-go.html

package main

import (
	"flag"
	"fmt"
	"strings"
	//"io/ioutil"
	"golang.org/x/net/html"
	"net/http"
)

var (
	VERSION    = "0.1-dev"
	BUILD_DATE = ""
)

var url = flag.String("url", "", "The url to get")
var version = flag.Bool("version", false, "print version information")

// Helper function to pull the  attribute from a Token
func getSrc(t html.Token) (ok bool, src string) {
	// Iterate over all of the Token's attributes until we find an "src"
	for _, a := range t.Attr {
		if a.Key == "src" {
			src = a.Val
			ok = true
		}
	}

	// "bare" return will return the variables (ok, href) as defined in
	// the function definition
	return
}

// Extract all http** links from a given webpage
func crawl(url string, ch chan string, chFinished chan bool) {
	resp, err := http.Get(url)

	defer func() {
		// Notify that we're done after this function
		chFinished <- true
	}()

	if err != nil {
		fmt.Println("ERROR: Failed to crawl \"" + url + "\"")
		return
	}

	b := resp.Body
	defer b.Close() // close Body when the function returns

	z := html.NewTokenizer(b)

	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			return
		case tt == html.SelfClosingTagToken:
			t := z.Token()

			// Check if the token is an <a> tag
			isAnchor := t.Data == "img"
			if !isAnchor {
				continue
			}

			// Extract the src value, if there is one
			ok, url := getSrc(t)
			if !ok {
				continue
			}

			// Make sure the url begines in http**
			hasProto := strings.Index(url, "http") == 0
			if hasProto {
				ch <- url
			}
		}
	}
}

func main() {
	flag.Parse()

	foundUrls := make(map[string]bool)

	// Channels
	chUrls := make(chan string)
	chFinished := make(chan bool)

	if *version {
		fmt.Printf("%v\nBuild: %v\n", VERSION, BUILD_DATE)
		return
	}

	fmt.Printf("go go gadgeto go !!!\n")

	go crawl(*url, chUrls, chFinished)

	// Subscribe to both channels
	for c := 0; c < 1; {
		select {
		case url := <-chUrls:
			foundUrls[url] = true
		case <-chFinished:
			c++
		}
	}

	// We're done! Print the results...

	fmt.Println("\nFound", len(foundUrls), "unique urls:\n")

	for url, _ := range foundUrls {
		fmt.Println(" - " + url)
	}

	close(chUrls)

	//	resp, err := http.Get(*url)
	//
	//	if err != nil {
	//		fmt.Print("Ooooh noooo - something bad happen *_*\n")
	//		return
	//	}
	//
	//	defer resp.Body.Close()
	//	body, err := ioutil.ReadAll(resp.Body)

	//	fmt.Printf("%s", body)
}
