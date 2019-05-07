package crawler

import (
	"time"
)

// external Crawler struct with config parameters
type Crawler struct {
	Timeout             time.Duration
	MaxDepth            int
	Workers             int
	Auth                *CrawlerAuth
	HashLoopCheck       bool
	FollowExternal      bool
	UserAgent           *string
	Retries             int
	SleepBetweenRetries time.Duration
}

// auth part of crawler config struct
type CrawlerAuth struct {
	Username string
	Password string
}

// struct returned to callback func for each url crawled with urls found
type FoundUrls struct {
	CrawlUrl  string
	FoundUrls []string
	Err       error
	Depth     int
}

// creates a new crawler object
func NewCrawler() (crawler *Crawler) {
	crawler = new(Crawler)
	crawler.Timeout = 60 * time.Second
	crawler.MaxDepth = -1
	crawler.Auth = nil
	crawler.Workers = 10
	crawler.HashLoopCheck = false
	crawler.FollowExternal = false
	crawler.UserAgent = nil
	crawler.Retries = 0
	crawler.SleepBetweenRetries = 0
	return
}

// run crawler: creates new crawl worker and runs the first job
func (c *Crawler) Crawl(baseUrl string, callbackFunc func(*FoundUrls)) {
	w := newCrawlWorker(c, baseUrl, callbackFunc)
	c.crawlInternal(w)
}

func (c *Crawler) crawlInternal(w crawlWorkerInterface) {
	w.addWorker()
	w.crawl("", 0)
	w.waitForWorkers()
}
