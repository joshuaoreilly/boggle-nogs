package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	fmt.Println("Hello, World!")
	// resp is of type Response:
	// https://pkg.go.dev/net/http#Response
	resp, err := http.Get("https://news.ycombinator.com/")
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
}
