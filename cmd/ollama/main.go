package main

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/xyproto/clickableai"
	"github.com/xyproto/ollamaclient/v2"
)

// Embed your files here
//
//go:embed extra.conf
var extraInHead string

//go:embed topics.conf
var initialTopics string

//go:embed githublogo.png
var githublogo []byte

//go:embed index.html
var indexHTML string

//go:embed robots.txt
var robots string

//go:embed markdown-it.min.js
var markdownJS []byte

const mainPrompt = "Generate correct, interesting and technical documentation about these keywords, in Markdown: "

var (
	currentKeywords = strings.Split(initialTopics, ",")
	keywordTrail    = []string{}
)

func main() {
	clickableai.InitTemplate(indexHTML)

	http.HandleFunc("/", handler)
	http.HandleFunc("/generate", generateHandler)
	http.HandleFunc("/githublogo.png", func(w http.ResponseWriter, r *http.Request) {
		clickableai.GithubLogoHandler(w, r, githublogo)
	})
	http.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		clickableai.RobotsHandler(w, r, robots)
	})
	http.HandleFunc("/markdown-it.min.js", func(w http.ResponseWriter, r *http.Request) {
		clickableai.MarkdownJSHandler(w, r, markdownJS)
	})

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	clickableai.Handler(w, r, currentKeywords, "", extraInHead)
}

func generateHandler(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("keyword")
	if keyword != "" {
		keywordTrail = append(keywordTrail, keyword)
	}

	markdown, newKeywords := generateMarkdownAndKeywords(keywordTrail)

	currentKeywords = newKeywords
	clickableai.Handler(w, r, newKeywords, markdown, extraInHead)
}

func generateMarkdownAndKeywords(trail []string) (string, []string) {
	prompt := mainPrompt + strings.Join(trail, " -> ")

	oc := ollamaclient.New()
	oc.Verbose = true

	if err := oc.PullIfNeeded(); err != nil {
		fmt.Println("Error:", err)
		return "Error: Could not pull model", nil
	}

	output, err := oc.GetOutput(prompt)
	if err != nil {
		fmt.Println("Error:", err)
		return "Error: Could not generate output", nil
	}

	newKeywords := []string{"Networking", "Databases", "Kubernetes"}
	followUpKeywordsString, err := oc.GetOutput("Generate 10 interesting follow-up keywords that relates to the following text:\n" + output + "\n\n" + "Only output the slice of strings, as Go code. No commentary!")
	if err == nil {
		followUpKeywordsString = strings.TrimPrefix(followUpKeywordsString, "```go")
		followUpKeywordsString = strings.TrimPrefix(followUpKeywordsString, "```")
		followUpKeywordsString = strings.TrimSuffix(followUpKeywordsString, "```")
		fields := strings.Split(followUpKeywordsString, ",")
		for i, field := range fields {
			fields[i] = clickableai.BetweenQuotes(field)
		}
		if len(fields) > 1 {
			newKeywords = fields
		}
	}

	return output, newKeywords
}
