//http://schier.co/blog/2015/04/26/a-simple-web-scraper-in-go.html

package main

import (
	"flag"
	"fmt"
	"strings"
	"golang.org/x/net/html"
	"net/http"
	"time"
	"io/ioutil"
	"bytes"
)

type download_statistic struct {
	url string
	response_time time.Duration
	response_size int
}

type global_statistic struct {
	total_response_time time.Duration
	total_response_size int
}

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
func fetch_main_url(url string) (map[string]bool, download_statistic) {
	foundUrls := make(map[string]bool)

	stat := download_statistic{url, 0, 0}

	t0 := time.Now()
	resp, err := http.Get(url)
	t1 := time.Now()

	stat.response_time = t1.Sub(t0)

	if err != nil {
		fmt.Println("ERROR: Failed to get input url \"" + url + "\"")
		return foundUrls, stat
	}


	body, err := ioutil.ReadAll(resp.Body)
	//defer b.Close() // close Body when the function returns
	if err != nil {
		fmt.Println("ERROR: Failed to read body for input url \"" + url + "\"")
		return foundUrls, stat
	}
	stat.response_size = len(body)

	fmt.Printf(" - [%s] %s %v %v\n", resp.Status, stat.url, stat.response_time, stat.response_size)
	//fmt.Printf("%s\n",body)

	//b := resp.Body
	//defer b.Close() // close Body when the function returns

	z := html.NewTokenizer(bytes.NewReader(body))

	for {
		tt := z.Next()
		//fmt.Printf(".")

		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			//fmt.Printf("   - end of doc\n")
			return foundUrls, stat
		case tt == html.SelfClosingTagToken:
			t := z.Token()

			//fmt.Printf("   - check %s\n",t.Data)

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
				foundUrls[url] = true
			}
		}
	}
}

func fetch_asset(url string, chStat chan download_statistic, chFinished chan bool) {

	stat := download_statistic{url, 0, 0}

	//fmt.Printf(" - try to get [%s]\n", url)
	t0 := time.Now()
	resp, err := http.Get(url)
	t1 := time.Now()

	stat.response_time = t1.Sub(t0)

	if err != nil {
		fmt.Println("ERROR: Failed to get link \"" + url + "\"")
		return
	}

	defer func() {
		// Notify that we're done after this function
		chFinished <- true
	}()

	//stat.response_size = resp.ContentLength

	b := resp.Body
	defer b.Close() // close Body when the function returns
	body, err := ioutil.ReadAll(resp.Body)

	stat.response_size = len(body)

	fmt.Printf(" - [%s] %s %v %v\n", resp.Status, stat.url, stat.response_time, stat.response_size)

	chStat <- stat
}

func main() {
	flag.Parse()

	//urls and global stats
	var foundUrls map[string]bool
	var main_url_stat download_statistic
	var gstat global_statistic

	// Channels
	chUrls := make(chan download_statistic)
	chFinished := make(chan bool)

	if *version {
		fmt.Printf("%v\nBuild: %v\n", VERSION, BUILD_DATE)
		return
	}

	t0 := time.Now()
	//Fetch the main url and get inner links
	foundUrls, main_url_stat = fetch_main_url(*url)
	gstat.total_response_time += main_url_stat.response_time
	gstat.total_response_size += main_url_stat.response_size

	for url,_ := range foundUrls {
		go fetch_asset(url, chUrls, chFinished)
	}

	// Subscribe to both channels
	for c := 0; c < len(foundUrls); {
		select {
		case stat := <-chUrls:
			gstat.total_response_time += stat.response_time
			gstat.total_response_size += stat.response_size
			foundUrls[stat.url] = true
		case <-chFinished:
			c++
		}
	}
	t1 := time.Now()

	// We're done! Print the results...

	fmt.Printf("The call took %v to run.\n", t1.Sub(t0))
	fmt.Printf("Cumulated time: %v.\n", gstat.total_response_time)
	fmt.Printf("Cumulated size: %v.\n", gstat.total_response_size)

	close(chUrls)

}
