package main

import (
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

var urls = []string{
       "https://www.python.org/doc/",
    "https://golang.org/doc/",
    "https://docs.djangoproject.com/en/stable/",
    "https://flask.palletsprojects.com/en/stable/",
    "https://fastapi.tiangolo.com/",
    "https://pandas.pydata.org/docs/",
    "https://numpy.org/doc/",
    "https://scikit-learn.org/stable/documentation.html",
    "https://matplotlib.org/stable/contents.html",
    "https://developer.mozilla.org/en-US/docs/Web",
    "https://news.ycombinator.com/",
    "https://www.theguardian.com/international",
    "https://www.reuters.com/",
    "https://www.cnn.com/world",
    "https://www.nytimes.com/international/",
}

func fetch(url string, wg *sync.WaitGroup, ch chan<- string) {
    defer wg.Done()
    resp, err := http.Get(url)
    if err != nil {
        return
    }
    defer resp.Body.Close()
    body, _ := ioutil.ReadAll(resp.Body)
    text := extractText(string(body))
    ch <- text
}

func extractText(htmlStr string) string {
    doc, _ := html.Parse(strings.NewReader(htmlStr))
    var f func(*html.Node) string
    f = func(n *html.Node) string {
        if n.Type == html.TextNode {
            return n.Data + " "
        }
        result := ""
        for c := n.FirstChild; c != nil; c = c.NextSibling {
            result += f(c)
	 }
	 return result
    }
    return f(doc)
}

func fetchURLs() chan string {
    ch := make(chan string, len(urls))
    var wg sync.WaitGroup
    for _, u := range urls {
        wg.Add(1)
	 go fetch(u, &wg, ch)
    }
    wg.Wait()
    close(ch)
    return ch
}

func main() {
    _ = fetchURLs()
}
