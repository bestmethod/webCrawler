# Crawler

## JSON Sitemap generator writted in Go

### For package use, see [here](crawler/README.md)

### Binary / executable

To grab a precompiled binary, head to [`gitlab->pipeline`](https://gitlab.com/bestmethod/webCrawler/pipelines) and download artifacts for the latest pipeline->job

#### Usage:

```
Usage: crawler [options] {url}

  -errors-to-stderr
    	print errors to stderr in addition to reporting them in json
  -follow-external
    	follow URLs external to crawl URL, without max-depth may run indefinitely
  -hash-check
    	check for loops by using checksums on each html file, may be slow
  -indent
    	indent output, or print each URL per line
  -max-depth int
    	max depth to crawl to, or -1 for unlimited (default -1)
  -password string
    	password for HTTP basic auth
  -retries int
    	on http GET failure, retry this many times
  -retry-sleep int
    	sleep this many milliseconds between retries (default 100)
  -timeout int
    	http GET timeout in seconds (default 60)
  -user-agent string
    	set a custom user-agent for the crawler
  -username string
    	username for HTTP basic auth
  -workers int
    	number of concurrent workers to crawl with (default 10)

Notes:
	* the crawler does not follow redirects
	* instead of passing username, you can set env variable CRAWLER_USER
	* instead of passing password, you can set env variable CRAWLER_PASS
```

#### Example:
```
$ crawler -indent -max-depth 1 -retries 3 -timeout 10 -workers 50 -hash-check https://glonek.uk
```

```json
[
{
	"CrawledUrl": "https://glonek.uk",
	"FoundUrls": [
		"https://glonek.uk#profile",
		"https://glonek.uk#cv",
		"https://glonek.uk#documents",
		"https://glonek.uk#contacts",
		"https://glonek.uk/static/robert-glonek-cv.pdf",
		"https://glonek.uk/static/robert-glonek-cv.doc",
		"https://github.com/bestmethod",
		"https://hub.docker.com/search/?isAutomated=0\u0026isOfficial=0\u0026page=1\u0026pullCount=0\u0026q=bestmethod\u0026starCount=0",
		"https://glonek.uk",
		"https://plus.google.com/+RobertGlonek",
		"https://twitter.com/bestmethodltd",
		"https://uk.linkedin.com/in/robert-glonek-3936a932"
	],
	"Depth": 0,
	"Error": ""
},
{
	"CrawledUrl": "https://glonek.uk#profile",
	"FoundUrls": null,
	"Depth": 1,
	"Error": "HashLoopCheck: https://glonek.uk"
},
{
	"CrawledUrl": "https://glonek.uk#documents",
	"FoundUrls": null,
	"Depth": 1,
	"Error": "HashLoopCheck: https://glonek.uk"
},
{
	"CrawledUrl": "https://glonek.uk#cv",
	"FoundUrls": null,
	"Depth": 1,
	"Error": "HashLoopCheck: https://glonek.uk"
},
{
	"CrawledUrl": "https://glonek.uk#contacts",
	"FoundUrls": null,
	"Depth": 1,
	"Error": "HashLoopCheck: https://glonek.uk"
},
]
```

#### Example with pipes
```
$ crawler -indent -max-depth 1 -retries 3 -timeout 10 -workers 50 -errors-to-stderr -hash-check https://glonek.uk > results.json

ERROR in `https://glonek.uk#documents`: HashLoopCheck: https://glonek.uk
ERROR in `https://glonek.uk#profile`: HashLoopCheck: https://glonek.uk
ERROR in `https://glonek.uk#cv`: HashLoopCheck: https://glonek.uk
ERROR in `https://glonek.uk#contacts`: HashLoopCheck: https://glonek.uk
```

#### Example auth, using a mix of env vars and params
```
$ CRAWLER_PASS="somepassword"
$ crawler -username robert -indent -max-depth 1 -retries 3 -timeout 10 -workers 50 -hash-check https://apps.glonek.uk > results.json
```
