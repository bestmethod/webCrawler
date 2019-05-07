package proof_of_concept

import (
	"errors"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

/*
handle -link-here (should remove last /something/ and replace with that -link-here link) for some reason

cleanup code, make package

allow to specify URL and to select thread count

allow connect timeout (client = http.Client(timeout...), client.Get

allow auth (if required)

error handling and output format

allow advanced loop check (hash page contents, if same hash found, it's a loop!)

should be main code doing this:
getting URLs from chan, go worker() each worker, getting results and parsing and launching more workers
worker() should be putting URLs find into chan
workers should not be calling self as memory usage inefficient

maxDepth parameter option
 */

type Urls struct {
	CrawlUrl  string
	FoundUrls []string
}

var crawlurls = []string{}

var count = make(chan bool, 100)

func extractHref(body io.Reader) []string {
	var links []string
	z := html.NewTokenizer(body)
	for {
		tt := z.Next()

		switch tt {
		case html.ErrorToken:
			return links
		case html.StartTagToken, html.EndTagToken:
			token := z.Token()
			if "a" == token.Data {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						links = append(links, attr.Val)
					}
				}
			}
		}
	}
}

var mutex = &sync.Mutex{}

func crowl(aurl string, origin string, callback func(urls *Urls, err error)) error {
	mutex.Lock()
	found := false
	for _, nurl := range crawlurls {
		if nurl == aurl {
			found = true
			break
		}
	}
	if found == true {
		mutex.Unlock()
		<-count
		return nil
	}
	crawlurls = append(crawlurls, aurl)
	mutex.Unlock()
	u := new(Urls)
	u.CrawlUrl = aurl
	resp, err := http.Get(aurl)
	if err != nil {
		callback(nil, errors.New(fmt.Sprintf("Error fetching URL %s: %s", aurl, err)))
		<-count
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		callback(nil, errors.New(fmt.Sprintf("ErrorCode received fetching URL %s: %d", aurl, resp.StatusCode)))
		<-count
		return nil
	}
	if len(resp.Header["Content-Type"]) > 0 {
		if strings.HasPrefix(resp.Header["Content-Type"][0], "text/html") == false {
			<-count
			return nil
		}
	}
	links := extractHref(resp.Body)
	// get found URLs and call Crowl on them
	for _, nurl := range links {
		if strings.HasPrefix(nurl,"http://") || strings.HasPrefix(nurl,"https://") {
			u.FoundUrls = append(u.FoundUrls, nurl)
		} else {
			ua, _ := url.Parse(aurl)
			rel, _ := ua.Parse(nurl)
			u.FoundUrls = append(u.FoundUrls, rel.String())
		}
	}
	callback(u, nil)
	<-count
	for _, aurl := range u.FoundUrls {
		if strings.HasPrefix(aurl, origin) {
			count <- true
			go crowl(aurl, origin, callback)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func Crowl(aurl string, callback func(urls *Urls, err error)) error {
	count <- true
	go crowl(aurl, aurl, callback)
	for len(count) > 0 {
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

func callback(urls *Urls, err error) {
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return
	}
	var urlString string
	for _, aurl := range urls.FoundUrls {
		urlString = fmt.Sprintf("%s\"%s\",", urlString, aurl)
	}
	//fmt.Printf("{\"CrawlUrl\": \"%s\",\"FoundUrls\": [%s]}\n", urls.CrawlUrl, urlString)
	fmt.Println(urls.CrawlUrl)
}

func main() {
	fmt.Println(time.Now().String())
	//fmt.Printf("{\"Crawl\": \"%s\",\"URLs\": [", "https://www.monzo.com")
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		//fmt.Println("]}")
		_, _ = fmt.Fprintln(os.Stderr, "Incomplete: interrupted by signal")
		os.Exit(1)
	}()
	err := Crowl("https://www.monzo.com/", callback)
	//fmt.Println("]}")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error Crawling: %s\n", err)
	}
	fmt.Println(time.Now().String())
}
