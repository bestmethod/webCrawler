package crawler

import (
	"errors"
	"fmt"
	"golang.org/x/net/html"
	"io"
)

// simple wrapper, cause I cannot be bothered to keep typing this
func makeError(format string, a ...interface{}) error {
	return errors.New(fmt.Sprintf(format, a...))
}

// as the name suggests, extracts <a href, and returns links
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
