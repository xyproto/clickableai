package main

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/xyproto/clickableai"
	"github.com/xyproto/env/v2"
	"github.com/xyproto/simpleflash"
)

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

const (
	TextModel          = "gemini-1.5-flash"
	MultiModalModel    = "gemini-1.0-pro-vision"
	MainPrompt         = "Generate a correct, concise, and technical Markdown document based on these keywords. No commentary: "
	TopicPrompt        = "Generate exactly 10 suitable topics based on these keywords and the following content. Output as a strict comma-separated list with no commentary: "
	GeneralTopicPrompt = "Generate 10 general keywords based on the following Markdown content. Output as a strict comma-separated list with no commentary: "
)

var (
	projectLocation = env.Str("PROJECT_LOCATION", "europe-north1")
	projectID       = env.Str("PROJECT_ID")
	sf              *simpleflash.SimpleFlash
)

func main() {
	if projectID == "" {
		log.Fatalln("Error: PROJECT_ID environment variable is not set.")
		return
	}

	var err error
	sf, err = simpleflash.New(TextModel, MultiModalModel, projectLocation, projectID, true)
	if err != nil {
		log.Fatalln("Error:", err)
		return
	}

	clickableai.InitTemplate(indexHTML)

	http.HandleFunc("/generate", generateHandler)
	http.HandleFunc("/generate_topics", generateTopicsHandler)
	http.HandleFunc("/githublogo.png", func(w http.ResponseWriter, r *http.Request) {
		clickableai.GithubLogoHandler(w, r, githublogo)
	})
	http.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		clickableai.RobotsHandler(w, r, robots)
	})
	http.HandleFunc("/markdown-it.min.js", func(w http.ResponseWriter, r *http.Request) {
		clickableai.MarkdownJSHandler(w, r, markdownJS)
	})
	http.HandleFunc("/", handler)

	port := env.Str("PORT", "8080")
	log.Println("Starting server on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	clickableai.Handler(w, r, strings.Split(initialTopics, ","), "", extraInHead)
}

func generateHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	keywords := r.Form["keywords"]
	markdown, _ := generateMarkdownAndKeywords(keywords)

	clickableai.Handler(w, r, keywords, markdown, extraInHead)
}

func generateTopicsHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	keywords := r.Form["keywords"]
	markdown := r.FormValue("markdown")

	newTopics := generateNewTopics(keywords, markdown)

	if len(newTopics) == 1 && strings.Contains(newTopics[0], "Error") {
		newTopics = generateGeneralTopics(markdown)
	}

	clickableai.Handler(w, r, newTopics, "", extraInHead)
}

func generateMarkdownAndKeywords(trail []string) (string, []string) {
	prompt := MainPrompt + strings.Join(trail, " -> ")

	temperature := 0.0
	output, err := sf.QueryGemini(prompt, &temperature, nil, nil)
	if err != nil {
		log.Println("Error:", err)
		return "Error: Could not generate output", nil
	}

	return output, nil
}

func generateNewTopics(keywords []string, markdown string) []string {
	prompt := TopicPrompt + strings.Join(keywords, ", ") + " | Content: " + markdown

	temperature := 0.5
	topicsOutput, err := sf.QueryGemini(prompt, &temperature, nil, nil)
	if err != nil {
		log.Println("Error:", err)
		return []string{"Error: Could not generate topics"}
	}

	return clickableai.ExtractAndShortenTopics(topicsOutput, keywords)
}

func generateGeneralTopics(markdown string) []string {
	prompt := GeneralTopicPrompt + markdown

	fmt.Printf("Generating general new topics for %d bytes of Markdown.\n", len(markdown))

	temperature := 0.5
	topicsOutput, err := sf.QueryGemini(prompt, &temperature, nil, nil)
	if err != nil {
		fmt.Println("Error:", err)
		return []string{"Error: Could not generate topics"}
	}

	return clickableai.ExtractAndShortenTopics(topicsOutput, []string{})
}
