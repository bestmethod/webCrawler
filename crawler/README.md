# Crawler

## URL link crawler written in go

### Summary

Crawler goes through URLs and their links, finding href in each text/html URL, and reporting these links per crawled URL to a callback function

#### Usage

`import "github.com/bestmethod/webCrawler/crawler"`

#### Example:

```go
package main

import "github.com/bestmethod/webCrawler/crawler"
import "time"
import "fmt"

func callback(urls *crawler.FoundUrls) {
	fmt.Printf("Depth:%d Error:%s URL:%s Found:%s\n", urls.Depth, urls.Err, urls.CrawlUrl, urls.FoundUrls)
}

func main() {
    c := crawler.NewCrawler()
    c.MaxDepth = 5
    c.Timeout = 10 * time.Second
    c.Crawl("https://example.org", callback)
}

```

#### Index

* [func NewCrawler() (crawler *Crawler)](#func-newcrawler)
* [type Crawler](#type-crawler)
  * [func (c *Crawler) Crawl(baseUrl string, callbackFunc func(*FoundUrls))](#func-c-crawler-crawlbaseurl-string-callbackfunc-funcfoundurls)
* [type CrawlerAuth](#type-crawlerauth)
* [type FoundUrls](#type-foundurls)

##### func NewCrawler

`func NewCrawler() (crawler *Crawler)`

Function create a basic [`Crawler`](#type-crawler) type, with default parameter settings. Should be called first before anything else, unless you set each parameter in the [`Crawler`](#type-crawler) type.

###### Example:

```go
c := crawler.NewCrawler()
c.MaxDepth = 5
c.Crawl("https://example.org", callback)
```

##### type Crawler

Basic Crawler configuration struct. Create default one using [`NewCrawler`](#func-newcrawler), and then overwrite the settings you wish.

```go
type Crawler struct {
    // Timeout for http connection 
    // default: 60s
	Timeout             time.Duration

    // maximum depth to crawl, -1 == unlimited 
    // default: -1
	MaxDepth            int

    // number of concurrent workers to use
    // default: 10
	Workers             int

    // should we use authentication
    // if not nil, will use credentials from CrawlerAuth
    // default: nil
	Auth                *CrawlerAuth

    // should we perform a loop/repeat check using hashes
    // may cause crawler to be slow
    // setting to true will cause the crawler to generate sha256 for each text/html file
    // default: false
	HashLoopCheck       bool

    // should we follow URLs external to the crawled URL
    // if set, while MaxDepth is -1, may result in crawling the whole internet
    // default: false
	FollowExternal      bool

    // set non-standard UserAgent
    // if nil, default from net/http package is used
    // default: nil
	UserAgent           *string

    // how many times to retry on statusCode other than 2xx
    // deault: 0
	Retries             int

    // if Retries != 0, how long to sleep between retries
    // default: 100ms
	SleepBetweenRetries time.Duration
}
```

##### func (c *Crawler) Crawl

`func (c *Crawler) Crawl(baseUrl string, callbackFunc func(*FoundUrls))`

Function starts a crawl and each time it finishes parsing a URL for all it's links, it calls a user-defined callbackFunc, handling the results in the format of [`Crawler.FoundUrls`](#type-foundurls)

###### Example:

```go
func callback(urls *crawler.FoundUrls) {
	fmt.Printf("Depth:%d Error:%s URL:%s Found:%s\n", urls.Depth, urls.Err, urls.CrawlUrl, urls.FoundUrls)
}

func main() {
    c := crawler.NewCrawler()
    c.MaxDepth = 5
    c.Crawl("https://example.org", callback)
}
```

##### type CrawlerAuth

Struct for HTTP basic auth for URL crawl. Create this and set [`Crawler.Auth`](#type-crawler) to it

```go
type CrawlerAuth struct {

	// username to pass to http.Get
	Username string

	// password to pass to http.Get
	Password string
}
```

##### type FoundUrls

Struct returned to callback function for each URL crawled with a list of links found on that URL

```go
type FoundUrls struct {
	
	// URL that was crawled / parsed - always set unless disaster happens
	CrawlUrl  string
	
	// list of links found while crawling the CrawlUrl, translated to absolute URLs
	// may be empty if no links found or error occurred
	FoundUrls []string
	
	// a list of any errors occurred while crawling and parsing the CrawlUrl
	// String error will be of format "error 1 text && error 2 text && ..."
	Err       error
	
	// crawl dept at which the CrawlUrl resides, relative to the origin crawl URL
	Depth     int
}
```
