package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

/*
TODO:
- Fix logging when behind proxy
- Nicer HTML/CSS for easier reading
- Add dates to post details?
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
var local bool

var logger log.Logger

var regexSiteLink = regexp.MustCompile(`(site=)`)
var regexNextPage = regexp.MustCompile(`(\/\?p=\d)`)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func errorHandler(w http.ResponseWriter, r *http.Request, reason string) {
	/*
		Catch-all error handler; incorrectly returns StatusNotFound in all cases
	*/
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, reason)
	logger.Println(reason)
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	/*
		Checks if the requested URL is valid/supported,
		gets news.ycombinator.com page,
		scrapes it,
		and sends the output.
	*/
	logger.Printf("Requested %s%s", domain, r.URL.RequestURI())

	if !(r.URL.Path == "/" ||
		regexNextPage.MatchString(r.URL.RawQuery) ||
		(r.URL.Path == "/from" &&
			regexSiteLink.MatchString(r.URL.RawQuery))) {
		errorHandler(w, r, fmt.Sprintf("404: Page %s not found", domain+r.URL.RequestURI()))
		return
	}

	body := readHtmlFromWebsite("https://news.ycombinator.com" + r.URL.RequestURI())
	posts, nextPageLink := parseHtml(body)
	stringBuilder := createHtml(posts, nextPageLink)
	page := stringBuilder.String()
	_, err := fmt.Fprint(w, page)
	check(err)
}

func readHtmlFromWebsite(url string) string {
	/*
		Get html from url
	*/
	resp, err := http.Get(url)
	check(err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	check(err)

	// 0644 is owner read and write, but not execute permissions
	err = os.WriteFile("hackernews.html", body, 0644)
	check(err)
	return string(body)
}

func parseHtml(body string) (posts []Post, nextPageLink string) {
	/*
		Iterate through HTML response from news.ycombinator.com for posts and the "more" link
	*/
	tokenizer := html.NewTokenizer(strings.NewReader(body))
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			// At the end of the document, but missing nextPageLink
			// Should probably handle this
			return
		} else if tokenType == html.StartTagToken {
			token := tokenizer.Token()
			if token.Data == "td" && strings.Contains(token.String(), "align=\"right\" valign=\"top\" class=\"title\"") {
				// Found a title
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
			} else if token.Data == "a" && len(token.Attr) > 1 && token.Attr[1].Val == "morelink" {
				nextPageLink = token.Attr[0].Val
				// we have everything we need
				return
			}
		}
	}
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
					p.title = "TITLE NOT FOUND"
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
					p.site = "SITE NOT FOUND"
				}
			} else if token.Data == "span" && token.Attr[0].Val == "score" {
				// score
				tokenType := tokenizer.Next()
				if tokenType == html.TextToken {
					token := tokenizer.Token()
					p.score = token.String()
				} else {
					p.score = "SCORE NOT FOUND"
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
					p.commentsLink = stringBuilder.String()
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

func createHtml(posts []Post, nextPageLink string) strings.Builder {
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
	stringBuilder.WriteString("<h1><a class=\"title\" href=\"/\">Boggle Nogs</a></h1>\n")
	stringBuilder.WriteString("<div class=\"posts\">\n")

	for _, post := range posts {
		stringBuilder.WriteString(fmt.Sprintf("<div class=\"left\"><span class=\"grey\">%s</span></div>\n", post.rank))
		stringBuilder.WriteString("<div class=\"right\">\n")
		stringBuilder.WriteString(fmt.Sprintf("<a class=\"black\" href=\"%s\">%s</a> ", post.titleLink, post.title))
		stringBuilder.WriteString(fmt.Sprintf("<span class=\"grey\">(</span><a class=\"grey\" href=\"/%s\">%s</a><span class=\"grey\">)</span>\n", post.siteLink, post.site))
		stringBuilder.WriteString("<br>\n")
		stringBuilder.WriteString(fmt.Sprintf("<span class=\"grey\">%s</span>\n", post.score))
		stringBuilder.WriteString(fmt.Sprintf("<a class=\"grey\" href=\"%s\">%s</a>\n", post.commentsLink, post.comments))
		stringBuilder.WriteString("</div>\n")
	}

	stringBuilder.WriteString("</div>\n")
	stringBuilder.WriteString(fmt.Sprintf("<a class=\"black\" href=\"/%s\">%s</a>\n", nextPageLink, "more"))
	stringBuilder.WriteString(string(foot))

	return stringBuilder
}

func main() {
	f, err := os.Create("log.log")
	check(err)
	defer f.Close()

	mw := io.MultiWriter(os.Stdout, f)
	logger.SetOutput(mw)
	logger.SetFlags(log.Ldate | log.Ltime)

	var domainFlag = flag.String("domain", "", "domain name of domain, if NOT behind proxy")
	var portFlag = flag.Int("port", 1616, "port to run boggle nogs on")
	var localFlag = flag.Bool("local", false, "if running on localhost")

	flag.Parse()

	domain = *domainFlag
	port = *portFlag
	local = *localFlag

	if local {
		domain = "http://localhost"
	}

	logger.Printf("Links will be generated with domain %s", domain)
	logger.Printf("Running on port %d", port)

	// match everything
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRequest)
	err = http.ListenAndServe(":"+fmt.Sprint(port), mux)
	check(err)
}
