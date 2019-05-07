package crawler

import (
	"bytes"
	"crypto/sha256"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// crawl worker struct, contains config and all states that are needed by the crawler
type crawlWorker struct {
	crawler        *Crawler
	baseUrl        string
	callbackFunc   func(*FoundUrls)
	crawledUrls    []*string
	mutex          *sync.Mutex
	crawledUrlHash map[string]*[]byte
	hashMutex      *sync.Mutex
	workers        chan int
	workerSync     sync.WaitGroup
}

type crawlWorkerInterface interface {
	crawl(crawlUrl string, depth int)
	crawlWork(crawlUrl string, depth int) (u *FoundUrls)
	doHttpRequest(crawlUrl string) (r *http.Response, err error)
	addWorker()
	waitForWorkers()
}

func (w *crawlWorker) addWorker() {
	w.workers<-1
	w.workerSync.Add(1)
}

func (w *crawlWorker) waitForWorkers() {
	w.workerSync.Wait()
}

// creates and returns new crawlWorker struct, setting basics in the struct
func newCrawlWorker(c *Crawler, baseUrl string, callbackFunc func(*FoundUrls)) (w *crawlWorker) {
	w = new(crawlWorker)
	w.crawler = c
	w.baseUrl = baseUrl
	w.callbackFunc = callbackFunc
	w.mutex = &sync.Mutex{}
	w.hashMutex = &sync.Mutex{}
	w.workers = make(chan int, c.Workers)
	w.crawledUrlHash = make(map[string]*[]byte)
	return
}

// crawl: runs the worker, parses the return, calls callback and calls self for each FoundUrl to keep crawling deeper
func (w *crawlWorker) crawl(crawlUrl string, depth int) {
	if crawlUrl == "" && depth == 0 {
		crawlUrl = w.baseUrl
	}
	u := w.crawlWork(crawlUrl, depth)
	if u != nil {
		w.callbackFunc(u)
		if w.crawler.MaxDepth < 0 || depth < w.crawler.MaxDepth {
			for _, aurl := range u.FoundUrls {
				if w.crawler.FollowExternal == true || strings.HasPrefix(aurl, w.baseUrl) {
					w.addWorker()
					go w.crawl(aurl, depth+1)
				}
			}
		}
	}
	w.workerSync.Done()
	return
}

// actual crawl worker, crawls the URL, fills the FoundUrls object and returns it
// may return NIL if output is to be ignored (URL was not text/html for example, or already crawled this URL)
func (w *crawlWorker) crawlWork(crawlUrl string, depth int) (u *FoundUrls) {

	// always create, set basics
	u = new(FoundUrls)
	u.CrawlUrl = crawlUrl
	u.Depth = depth

	// signal on chan once we are done
	defer func() { <-w.workers }()

	// find if we crawled this before, if so exit, if not add to list of 'crawled this'
	w.mutex.Lock()
	for _, nurl := range w.crawledUrls {
		if *nurl == crawlUrl {
			w.mutex.Unlock()
			return nil
		}
	}
	w.crawledUrls = append(w.crawledUrls, &crawlUrl)
	w.mutex.Unlock()

	// handle HTTP request
	// handles retries and sleep between retries
	var resp *http.Response
	var err error
	for retries := 0; retries <= w.crawler.Retries; retries += 1 {
		resp, err = w.doHttpRequest(crawlUrl)
		if err != nil {
			if retries == w.crawler.Retries {
				u.Err = makeError("doHttpRequest: %s", err)
				return
			} else if w.crawler.SleepBetweenRetries > 0 {
				time.Sleep(w.crawler.SleepBetweenRetries)
			}
		} else {
			break
		}
	}
	defer resp.Body.Close()

	// if content-type header exists, and it's NOT text/html, simply return nil, not a HTML file
	if len(resp.Header["Content-Type"]) > 0 {
		if strings.HasPrefix(resp.Header["Content-Type"][0], "text/html") == false {
			return nil
		}
	}

	// if HashLoopCheck is true, handle checking if hash was already crawled, set error and return if yes
	// otherwise, add to hash list
	var respBody io.Reader
	if w.crawler.HashLoopCheck == true {
		hasher := sha256.New()
		respBody = io.TeeReader(resp.Body, hasher)
		sum := hasher.Sum(nil)
		w.hashMutex.Lock()
		for hashUrl, hash := range w.crawledUrlHash {
			if bytes.Compare(*hash, sum) == 0 {
				w.hashMutex.Unlock()
				u.Err = makeError("HashLoopCheck: %s", hashUrl)
				return
			}
		}
		w.crawledUrlHash[crawlUrl] = &sum
		w.hashMutex.Unlock()
	} else {
		respBody = resp.Body
	}

	// extract '<a href=' links, parrse them and add to list of FoundUrls
	// here if we have an issue parsing the URL, we will set an error, but will not return without finishing parsing
	links := extractHref(respBody)
	for _, link := range links {
		if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
			u.FoundUrls = append(u.FoundUrls, link)
		} else {
			linkUrl, err := url.Parse(crawlUrl)
			if err != nil {
				if u.Err == nil {
					u.Err = makeError("url.Parse: %s", err)
				} else {
					u.Err = makeError("%s && url.Parse: %s", u.Err, err)
				}
				continue
			}
			rel, err := linkUrl.Parse(link)
			if err != nil {
				if u.Err == nil {
					u.Err = makeError("url.Parse.Parse(%s): %s", link, err)
				} else {
					u.Err = makeError("%s && url.Parse.Parse(%s): %s", u.Err, link, err)
				}
				continue
			}
			u.FoundUrls = append(u.FoundUrls, rel.String())
		}
	}

	// success!!!
	return
}

// handle actual HTTP call, return response or error, calling function can deal with the retries, if any
func (w *crawlWorker) doHttpRequest(crawlUrl string) (r *http.Response, err error) {
	// create http client, configure it and call to make a GET request
	client := new(http.Client)
	client.Timeout = w.crawler.Timeout
	var req *http.Request
	req, err = http.NewRequest("GET", crawlUrl, nil)
	if err != nil {
		err = makeError("http.NewRequest: %s", err)
		return
	}
	if w.crawler.UserAgent != nil {
		req.Header.Set("User-Agent", *w.crawler.UserAgent)
	}
	if w.crawler.Auth != nil {
		req.SetBasicAuth(w.crawler.Auth.Username, w.crawler.Auth.Password)
	}
	r, err = client.Do(req)
	if err != nil {
		err = makeError("http.Do: %s", err)
		return
	}

	// handle statusCode other than success
	if r.StatusCode < 200 || r.StatusCode >= 300 {
		err = makeError("statusCode: %d", r.StatusCode)
		return
	}

	// success!
	return
}
