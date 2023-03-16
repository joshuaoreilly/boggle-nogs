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
	rank         string
	titleLink    string
	title        string
	score        string
	commentsLink string
	comments     string
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

func readHtmlFile() string {
	fi, _ := os.Open("hackernews.html")
	body, _ := io.ReadAll(fi)
	return string(body)
}

func parsePost(p *Post, tokenizer *html.Tokenizer) {
	commentsLinkFound := false
	titleFound := false
	for !commentsLinkFound {
		tokenType := tokenizer.Next()
		if tokenType == html.StartTagToken {
			token := tokenizer.Token()
			if token.Data == "span" && token.Attr[0].Val == "rank" {
				// rank
				tokenType := tokenizer.Next()
				if tokenType == html.TextToken {
					token := tokenizer.Token()
					p.rank = strings.Trim(token.String(), ".")
					fmt.Printf("Rank: %s\n", p.rank)
				} else {
					// It should be text, we have an error
					panic("No Rank found")
				}
			} else if !titleFound && token.Data == "a" && token.Attr[0].Key == "href" {
				// titleLink and title
				// Ask HN posts
				tokenVal := token.Attr[0].Val
				var titleLink string
				if len(tokenVal) > 7 && tokenVal[:8] == "item?id=" {
					// Ask HN/Show HN/Launch HN posts
					var stringBuilder strings.Builder
					stringBuilder.WriteString("https://news.ycombinator.com/")
					stringBuilder.WriteString(tokenVal)
					titleLink = stringBuilder.String()
				} else {
					titleLink = tokenVal
				}
				p.titleLink = titleLink
				fmt.Printf("Title Link: %s\n", p.titleLink)
				// title should immediately follow title link
				tokenType := tokenizer.Next()
				if tokenType == html.TextToken {
					token := tokenizer.Token()
					p.title = token.String()
					fmt.Printf("Title: %s\n", p.title)
				} else {
					// It should be text, we have an error
					panic("No Rank found")
				}
				titleFound = true
			} else if token.Data == "span" && token.Attr[0].Val == "score" {
				// score
				tokenType := tokenizer.Next()
				if tokenType == html.TextToken {
					token := tokenizer.Token()
					p.score = strings.Trim(token.String(), " points")
					fmt.Printf("Score: %s\n", p.score)
				} else {
					// It should be text, we have an error
					panic("No Rank found")
				}
			} else if titleFound && token.Data == "a" && token.Attr[0].Key == "href" {
				// comment link and comment
				// could be time since posting link as well, so we need to add addition criteria based
				// on the TextToken which follows
				linkVal := token.Attr[0].Val
				tokenType := tokenizer.Next()
				token := tokenizer.Token()
				if tokenType == html.TextToken && (strings.Contains(token.String(), "comment") || strings.Contains(token.String(), "discuss")) {
					var stringBuilder strings.Builder
					stringBuilder.WriteString("https://news.ycombinator.com/")
					stringBuilder.WriteString(linkVal)
					commentsLink := stringBuilder.String()
					p.commentsLink = commentsLink
					p.comments = token.String()
					fmt.Printf("Comments link: %s\n", p.commentsLink)
					fmt.Printf("Comments: %s\n", p.comments)
					commentsLinkFound = true
				}
			} else if !commentsLinkFound && token.Data == "tr" && len(token.Attr) > 0 && strings.Contains(token.String(), "spacer") { //token.Data == "tr" { //&& len(token.Attr) > 0 && token.Attr[0].Key == "spacer" {
				// found a new post without finishing the last one, probably corresponds to the posts of YC
				// companies that are hiring (they don't have comments or scores), or a post with no comments yet
				// we'll throw it out when iterating over posts
				// TODO: handle possible failure to find rank, title, etc.
				commentsLinkFound = true
			}
		}
	}
}

func printPosts(posts []Post) {
	for _, post := range posts {
		fmt.Printf("Rank: %s\nTitle: %s\nTitle Link: %s\nScore: %s\nComments: %s\nComments Link: %s\n",
			post.rank,
			post.title,
			post.titleLink,
			post.score,
			post.comments,
			post.commentsLink,
		)
	}
}

func parseHtml(body string) (posts []Post, nextPageLink string) {
	tokenizer := html.NewTokenizer(strings.NewReader(body))
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			// At the end of the document, but missing nextPageLink
			return
		} else if tokenType == html.StartTagToken {
			token := tokenizer.Token()
			// Found a title
			if token.Data == "td" && strings.Contains(token.String(), "align=\"right\" valign=\"top\" class=\"title\"") {
				p := Post{
					rank:         "0",
					titleLink:    "titleLink",
					title:        "title",
					score:        "0",
					commentsLink: "commentsLink",
					comments:     "comments",
				}
				// TODO: handle possible failure to find rank, title, etc.
				parsePost(&p, tokenizer)
				posts = append(posts, p)
			}
			if token.Data == "a" && len(token.Attr) > 1 && token.Attr[1].Val == "morelink" {
				nextPageLink = token.Attr[0].Val
				// we have everything we need
				return
			}
		}
	}
}

func main() {
	//body := readHtmlFromWebsite("https://news.ycombinator.com/")
	body := readHtmlFile() // for testing
	posts, nextPageLink := parseHtml(body)
	printPosts(posts)
	fmt.Println(nextPageLink)
	fmt.Printf("Number of posts found: %d\n", len(posts))
}
