package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/html"
)

type Post struct {
	rank         int
	title        string
	titleLink    string
	score        int
	comments     string
	commentsLink string
}

func readHtmlFromWebsite(url string) string {
	// resp is of type Response:
	// https://pkg.go.dev/net/http#Response
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	// resp.Body is of type io.ReadCloser (interface for Read() and Close() method)
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		panic(err)
	}

	// 0644 is owner read and write, but not execute permissions
	err = os.WriteFile("hackernews.html", body, 0644)
	if err != nil {
		panic(err)
	}

	return string(body)
}

func parsePost(p *Post, tokenizer *html.Tokenizer) {
	commentsLinkFound := false
	for !commentsLinkFound {
		tokenType := tokenizer.Next()
		if tokenType == html.StartTagToken {
			token := tokenizer.Token()
			if strings.Contains(token.String(), "span class=\"rank\"") {
				tokenType := tokenizer.Next()
				if tokenType == html.TextToken {
					token := tokenizer.Token()
					fmt.Println(token.String())
					commentsLinkFound = true
				} else {
					// It should be text, we have an error
					panic("No Rank found")
				}
			}
		}
	}
}

func parseHtml(body string) {
	var posts []Post
	tokenizer := html.NewTokenizer(strings.NewReader(body))
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			return
		} else if tokenType == html.StartTagToken {
			token := tokenizer.Token()
			// Found a title
			if token.Data == "td" && strings.Contains(token.String(), "align=\"right\" valign=\"top\" class=\"title\"") {
				p := Post{
					rank:         0,
					title:        "title",
					titleLink:    "titleLink",
					comments:     "comments",
					commentsLink: "commentsLink",
				}
				parsePost(&p, tokenizer)
				posts = append(posts, p)
			}
		}
	}
}

func main() {
	body := readHtmlFromWebsite("https://news.ycombinator.com/")
	parseHtml(body)
}
