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
    crawlWorkCheckList(crawlUrl string) (doWork bool)
	crawlWorkGetRetry(crawlUrl string) (resp *http.Response, err error)
	crawlWorkHashLoopCheck(crawlUrl string, resp *http.Response) (respBody io.Reader, err error)
	crawlWorkParseUrls(crawlUrl string, link string) (foundUrl string, err error)
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
				if w.crawler.FollowExternal == true || strings.HasPrefix(*aurl, w.baseUrl) {
					w.addWorker()
					go w.crawl(*aurl, depth+1)
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
	if w.crawlWorkCheckList(crawlUrl) == false {
		return nil
	}

	// handle HTTP request
	// handles retries and sleep between retries
	resp, err := w.crawlWorkGetRetry(crawlUrl)
	if err != nil {
		u.Err = err
		return
	}
	defer func() { _ = resp.Body.Close() }()

	// if content-type header exists, and it's NOT text/html, simply return nil, not a HTML file
	if len(resp.Header["Content-Type"]) > 0 {
		if strings.HasPrefix(resp.Header["Content-Type"][0], "text/html") == false {
			return nil
		}
	}

	// if HashLoopCheck is true, handle checking if hash was already crawled, set error and return if yes
	// otherwise, add to hash list
	respBody, err := w.crawlWorkHashLoopCheck(crawlUrl, resp)
	if err != nil {
		u.Err = err
		return
	}

	// extract '<a href=' links, parrse them and add to list of FoundUrls
	// here if we have an issue parsing the URL, we will set an error, but will not return without finishing parsing
	links := extractHref(respBody)
	for _, link := range links {
		foundUrl, err := w.crawlWorkParseUrls(crawlUrl, link)
		if err != nil {
			if u.Err == nil {
				u.Err = err
			} else {
				u.Err = makeError("%s && %s", u.Err, err)
			}
			continue
		}
		u.FoundUrls = append(u.FoundUrls, &foundUrl)
	}

	// success!!!
	return
}

func (w *crawlWorker) crawlWorkParseUrls(crawlUrl string, link string) (foundUrl string, err error) {
	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		foundUrl = link
	} else {
		linkUrl, errP := url.Parse(crawlUrl)
		if errP != nil {
			err = makeError("url.Parse: %s", errP)
			return
		}
		rel, errP := linkUrl.Parse(link)
		if errP != nil {
			err = makeError("url.Parse.Parse(%s): %s", link, errP)
			return
		}
		foundUrl = rel.String()
	}
	return
}

func (w *crawlWorker) crawlWorkHashLoopCheck(crawlUrl string, resp *http.Response) (respBody io.Reader, err error) {
	if w.crawler.HashLoopCheck == true {
		hasher := sha256.New()
		respBody = io.TeeReader(resp.Body, hasher)
		sum := hasher.Sum(nil)
		w.hashMutex.Lock()
		defer w.hashMutex.Unlock()
		for hashUrl, hash := range w.crawledUrlHash {
			if bytes.Compare(*hash, sum) == 0 {
				err = makeError("HashLoopCheck: %s", hashUrl)
				return
			}
		}
		w.crawledUrlHash[crawlUrl] = &sum
	} else {
		respBody = resp.Body
	}
	return
}

func (w *crawlWorker) crawlWorkGetRetry(crawlUrl string) (resp *http.Response, err error) {
	for retries := 0; retries <= w.crawler.Retries; retries += 1 {
		resp, err = w.doHttpRequest(crawlUrl)
		if err != nil {
			if retries == w.crawler.Retries {
				err = makeError("doHttpRequest: %s", err)
				return
			} else if w.crawler.SleepBetweenRetries > 0 {
				time.Sleep(w.crawler.SleepBetweenRetries)
			}
		} else {
			break
		}
	}
	return
}

func (w *crawlWorker) crawlWorkCheckList(crawlUrl string) (doWork bool) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	for _, nurl := range w.crawledUrls {
		if *nurl == crawlUrl {
			return false
		}
	}
	w.crawledUrls = append(w.crawledUrls, &crawlUrl)
	return true
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
