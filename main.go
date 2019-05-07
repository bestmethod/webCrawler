package main

// TODO: tests

import (
	"./crawler"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"
)

// will be copying output in callback to this before parsing to json - json.Marshall doesn't handle error type
type JsonOutput struct {
	CrawledUrl string
	FoundUrls  []string
	Depth      int
	Error      string
}

// struct for callback method, to pass arguments to callback
type Callback struct {
	indent bool
	errStderr bool
}

// callback method, called from crawler
// received crawler.FoundUrls, parses, prints json
func (c *Callback) callback(u *crawler.FoundUrls) {
	nu := new(JsonOutput)
	nu.CrawledUrl = u.CrawlUrl
	nu.FoundUrls = u.FoundUrls
	nu.Depth = u.Depth
	if u.Err != nil {
		nu.Error = u.Err.Error()
		if c.errStderr == true {
			_, _ = fmt.Fprintf(os.Stderr,"ERROR in `%s`: %s\n", u.CrawlUrl, u.Err)
		}
	}
	var b []byte
	var err error
	if c.indent == true {
		b, err = json.MarshalIndent(nu, "", "\t")
	} else {
		b, err = json.Marshal(nu)
	}
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Could not make json from output: %s\n", err)
		return
	}
	fmt.Printf("%s,\n", string(b))
}

// entrypoint
// parses command line arguments, sets handler for SIGINT, print json '[]' and runs crawler
func main() {
	// parse command line arguments
	errStdrr := flag.Bool("errors-to-stderr", false, "print errors to stderr in addition to reporting them in json")
	indent := flag.Bool("indent", false, "indent output, or print each URL per line")
	retries := flag.Int("retries", 0, "on http GET failure, retry this many times")
	retrySleep := flag.Int("retry-sleep", 100, "sleep this many milliseconds between retries")
	timeout := flag.Int("timeout", 60, "http GET timeout in seconds")
	maxDepth := flag.Int("max-depth", -1, "max depth to crawl to, or -1 for unlimited")
	followExternal := flag.Bool("follow-external", false, "follow URLs external to crawl URL, without max-depth may run indefinitely")
	workers := flag.Int("workers", 10, "number of concurrent workers to crawl with")
	hashCheck := flag.Bool("hash-check", false, "check for loops by using checksums on each html file, may be slow")
	username := flag.String("username", "", "username for HTTP basic auth")
	password := flag.String("password", "", "password for HTTP basic auth")
	useragent := flag.String("user-agent", "", "set a custom user-agent for the crawler")
	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "Usage: %s [options] {url}\n\n", os.Args[0])
		flag.PrintDefaults()
		_, _ = fmt.Fprintf(os.Stderr, "\nNotes:\n\t* the crawler does not follow redirects\n\t* instead of passing username, you can set env variable CRAWLER_USER\n\t* instead of passing password, you can set env variable CRAWLER_PASS\n\n")
	}
	flag.Parse()

	user := os.Getenv("CRAWLER_USER")
	pass := os.Getenv("CRAWLER_PASS")
	if username != nil && *username != "" {
		user = *username
	}
	if password != nil && *password != "" {
		pass = *password
	}
	// parase arguments to crawler struct
	c := crawler.NewCrawler()
	c.HashLoopCheck = *hashCheck
	c.Workers = *workers
	c.MaxDepth = *maxDepth
	c.FollowExternal = *followExternal
	c.Retries = *retries
	c.SleepBetweenRetries = time.Duration(*retrySleep) * time.Millisecond
	if useragent != nil && *useragent != "" {
		c.UserAgent = useragent
	}
	c.Timeout = time.Duration(*timeout) * time.Second
	if user != "" || pass != "" {
		cAuth := crawler.CrawlerAuth{Username: user, Password: pass}
		c.Auth = &cAuth
	}
	tail := flag.Args()
	if len(tail) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "missing url")
		flag.Usage()
		os.Exit(2)
	}
	if len(tail) > 1 {
		_, _ = fmt.Fprintf(os.Stderr, "only one URL allowed, too many arguments: %s\n", tail)
		flag.Usage()
		os.Exit(2)
	}

	// print json start/end, setup signal handler and run crawler
	fmt.Println("[")
	s := make(chan os.Signal)
	signal.Notify(s, os.Interrupt)
	go func() {
		<-s
		fmt.Println("]")
		_, _ = fmt.Fprintln(os.Stderr, "Incomplete: interrupted by signal")
		os.Exit(1)
	}()
	cb := new(Callback)
	cb.indent = *indent
	cb.errStderr = *errStdrr
	c.Crawl(tail[0], cb.callback)
	fmt.Println("]")
}
