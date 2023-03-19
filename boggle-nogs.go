package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

/*
TODO:
- Add dates to post details?
- Properly handle errors (don't just murder the program)
- Remove unecessary comments
- Figure out whether to defer or actively close
- Use firebase api instead of scraping website
- Add block-list support
*/

type Post struct {
	rank         string
	titleLink    string
	title        string
	siteLink     string
	site         string
	score        string
	commentsLink string
	comments     string
}

var domain string
var port int

var regexSiteLink = regexp.MustCompile(`(site=)`)
var regexNextPage = regexp.MustCompile(`(\/\?p=\d)`)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func readHtmlFromWebsite(url string) string {
	// resp is of type Response:
	// https://pkg.go.dev/net/http#Response
	resp, err := http.Get(url)
	check(err)

	defer resp.Body.Close()

	// resp.Body is of type io.ReadCloser (interface for Read() and Close() method)
	body, err := io.ReadAll(resp.Body)
	check(err)

	// 0644 is owner read and write, but not execute permissions
	err = os.WriteFile("hackernews.html", body, 0644)
	check(err)

	return string(body)
}

func readHtmlFile() string {
	fi, _ := os.Open("hackernews.html")
	defer fi.Close()
	body, _ := io.ReadAll(fi)
	return string(body)
}

func createHtml(domain string, port int, posts []Post, nextPageLink string) strings.Builder {
	if domain == "http://localhost" {
		domain = domain + ":" + fmt.Sprint(port)
	}
	fh, err := os.Open("head.html")
	check(err)
	defer fh.Close()
	ff, err := os.Open("foot.html")
	check(err)
	defer ff.Close()
	head, _ := io.ReadAll(fh)
	foot, _ := io.ReadAll(ff)
	var stringBuilder strings.Builder
	stringBuilder.WriteString(string(head))
	stringBuilder.WriteString(fmt.Sprintf("<h1><a href=\"%s\">Boggle Nogs</a></h1>\n", domain))
	stringBuilder.WriteString("<div class=\"posts\">\n")
	for _, post := range posts {
		stringBuilder.WriteString(fmt.Sprintf("<div class=\"left\">%s</div>\n", post.rank))
		stringBuilder.WriteString("<div class=\"right\">\n")
		stringBuilder.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a> ", post.titleLink, post.title))
		stringBuilder.WriteString(fmt.Sprintf("(<a href=\"%s/%s\">%s</a>)\n", domain, post.siteLink, post.site))
		stringBuilder.WriteString("<br>\n")
		stringBuilder.WriteString(fmt.Sprintf("%s\n", post.score))
		stringBuilder.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a>\n", post.commentsLink, post.comments))
		stringBuilder.WriteString("</div>\n")
	}
	stringBuilder.WriteString("</div>\n")
	stringBuilder.WriteString(fmt.Sprintf("<a href=\"%s/%s\">%s</a>\n", domain, nextPageLink, "more"))
	stringBuilder.WriteString(string(foot))
	return stringBuilder
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
				// title should immediately follow title link
				tokenType := tokenizer.Next()
				if tokenType == html.TextToken {
					token := tokenizer.Token()
					p.title = token.String()
				} else {
					// It should be text, we have an error
					panic("No Title found")
				}
				titleFound = true
			} else if token.Data == "a" && strings.Contains(token.Attr[0].Val, "from?site=") {
				// site link
				p.siteLink = token.Attr[0].Val
				// site should follow span element, which follows site link
				_ = tokenizer.Next()
				tokenType := tokenizer.Next()
				if tokenType == html.TextToken {
					token := tokenizer.Token()
					p.site = token.String()
				} else {
					// It should be text, we have an error
					panic("No Site found")
				}
			} else if token.Data == "span" && token.Attr[0].Val == "score" {
				// score
				tokenType := tokenizer.Next()
				if tokenType == html.TextToken {
					token := tokenizer.Token()
					p.score = token.String()
				} else {
					// It should be text, we have an error
					panic("No Score found")
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
					commentsLinkFound = true
				}
			} else if !commentsLinkFound && token.Data == "tr" && len(token.Attr) > 0 && strings.Contains(token.String(), "spacer") {
				// found a new post without finishing the last one, probably corresponds to the posts of YC
				// companies that are hiring (they don't have comments or scores)
				commentsLinkFound = true
			}
		}
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
					rank:         "",
					titleLink:    "",
					title:        "",
					siteLink:     "",
					site:         "",
					score:        "",
					commentsLink: "",
					comments:     "",
				}
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

func errorHandler(w http.ResponseWriter, r *http.Request, reason string) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, reason)
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	if !(r.URL.Path == "/" ||
		regexNextPage.MatchString(r.URL.RawQuery) ||
		(r.URL.Path == "/from" &&
			regexSiteLink.MatchString(r.URL.RawQuery))) {
		errorHandler(w, r, "404: Page not found")
		return
	}
	fmt.Print("https://news.ycombinator.com" + r.URL.Path + "\n")
	var urlEnd string
	if r.URL.RawQuery != "" {
		// next page, website articles, etc.
		urlEnd = r.URL.Path + "?" + r.URL.RawQuery
	} else {
		// base URL
		urlEnd = r.URL.Path
	}
	body := readHtmlFromWebsite("https://news.ycombinator.com" + urlEnd)
	fmt.Println("https://news.ycombinator.com" + urlEnd)
	//body := readHtmlFile() // for testing
	posts, nextPageLink := parseHtml(body)
	stringBuilder := createHtml(domain, port, posts, nextPageLink)
	page := stringBuilder.String()
	_, e := fmt.Fprint(w, page)
	check(e)
	// purely for debugging purposes
	f, err := os.Create("output.html")
	check(err)
	f.WriteString(page)
	f.Close()
}

func main() {
	var domainFlag = flag.String("domain", "http://localhost", "domain name of domain")
	var portFlag = flag.Int("port", 1616, "port to run boggle nogs on")
	flag.Parse()
	domain = *domainFlag
	port = *portFlag

	// match everything
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRequest)
	e := http.ListenAndServe(":"+fmt.Sprint(port), mux)
	check(e)
}
