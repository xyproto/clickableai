package main

import (
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/simpleflash"
)

//go:embed extra.conf
var extraInHead string // extra code that goes into <head>

//go:embed topics.conf
var initialTopics string // comma separated list of initial topics

//go:embed githublogo.png
var githublogo []byte

//go:embed index.html
var indexHTML string

//go:embed robots.txt
var robots string

//go:embed markdown-it.min.js
var markdownJS []byte

// Parse the indexHTML template and store it in "tmpl", or panic
var tmpl = template.Must(template.New("index").Parse(indexHTML))

type PageData struct {
	InitialTopics []string
	ExtraInHead   template.HTML
}

const (
	textModel          = "gemini-1.5-flash"
	multiModalModel    = "gemini-1.0-pro-vision"
	mainPrompt         = "Generate a correct, concise, and technical Markdown document based on these keywords. No commentary: "
	topicPrompt        = "Generate exactly 10 suitable topics based on these keywords and the following content. Output as a strict comma-separated list with no commentary: "
	generalTopicPrompt = "Generate 10 general keywords based on the following Markdown content. Output as a strict comma-separated list with no commentary: "
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
	sf, err = simpleflash.New(textModel, multiModalModel, projectLocation, projectID, true)
	if err != nil {
		log.Fatalln("Error:", err)
		return
	}

	// Serve the embedded resources
	http.HandleFunc("/generate", generateHandler)
	http.HandleFunc("/generate_topics", generateTopicsHandler)
	http.HandleFunc("/githublogo.png", githubLogoHandler)
	http.HandleFunc("/robots.txt", robotsHandler)
	http.HandleFunc("/markdown-it.min.js", markdownJSHandler)

	http.HandleFunc("/", handler)

	port := env.Str("PORT", "8080")
	log.Println("Starting server on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// robotsHandler serves the embedded robots.txt file
func robotsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(robots))
}

// githubLogoHandler serves the embedded githublogo.png file
func githubLogoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Write(githublogo)
}

// markdownJSHandler serves the embedded markdown-it.min.js file
func markdownJSHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Write(markdownJS)
}

func handler(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		InitialTopics: strings.Split(initialTopics, ","),
		ExtraInHead:   template.HTML(extraInHead),
	}

	// Execute the template and handle errors carefully
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing template: %s\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func generateHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	keywords := r.Form["keywords"]
	markdown, _ := generateMarkdownAndKeywords(keywords)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"markdown": %q}`, markdown)
}

func generateTopicsHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	keywords := r.Form["keywords"]
	markdown := r.FormValue("markdown")

	newTopics := generateNewTopics(keywords, markdown)

	// If new topics could not be generated, use a more general prompt
	if len(newTopics) == 1 && strings.Contains(newTopics[0], "Error") {
		newTopics = generateGeneralTopics(markdown)
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"topics": ["%s"]}`, strings.Join(newTopics, `","`))
}

func generateMarkdownAndKeywords(trail []string) (string, []string) {
	prompt := mainPrompt + strings.Join(trail, " -> ")

	temperature := 0.0
	output, err := sf.QueryGemini(prompt, &temperature, nil, nil)
	if err != nil {
		log.Println("Error:", err)
		return "Error: Could not generate output", nil
	}

	return output, nil
}

func generateNewTopics(keywords []string, markdown string) []string {
	prompt := topicPrompt + strings.Join(keywords, ", ") + " | Content: " + markdown

	temperature := 0.5
	topicsOutput, err := sf.QueryGemini(prompt, &temperature, nil, nil)
	if err != nil {
		log.Println("Error:", err)
		return []string{"Error: Could not generate topics"}
	}

	topics := extractAndShortenTopics(topicsOutput, keywords)

	return topics
}

func generateGeneralTopics(markdown string) []string {
	prompt := generalTopicPrompt + markdown

	fmt.Printf("Generating general new topics for %d bytes of Markdown.\n", len(markdown))

	temperature := 0.5
	topicsOutput, err := sf.QueryGemini(prompt, &temperature, nil, nil)
	if err != nil {
		fmt.Println("Error:", err)
		return []string{"Error: Could not generate topics"}
	}

	topics := extractAndShortenTopics(topicsOutput, []string{})

	return topics
}

// extractAndShortenTopics processes the output to remove redundant phrases and shorten topics to 1-2 words.
func extractAndShortenTopics(output string, keywords []string) []string {
	re := regexp.MustCompile(`(?:^|\s|,)([a-zA-Z0-9\-\_ ]{1,20})(?:,|\s|$)`)
	matches := re.FindAllString(output, -1)

	if len(matches) == 0 {
		log.Println("No valid topics found in the output")
		return []string{"Error: No valid topics found"}
	}

	topics := []string{}
	for _, match := range matches {
		topic := strings.TrimSpace(match)
		for _, keyword := range keywords {
			if strings.Contains(strings.ToLower(topic), strings.ToLower(keyword)) {
				topic = strings.Replace(topic, keyword, "", -1)
				topic = strings.TrimSpace(topic)
			}
		}
		topic = shortenToTwoWords(topic)
		topic = removeStrayCommas(topic)
		if isValidTopic(topic) && !contains(topics, topic) {
			topics = append(topics, topic)
		}
	}

	if len(topics) > 10 {
		topics = topics[:10]
	}

	return topics
}

func shortenToTwoWords(topic string) string {
	words := strings.Fields(topic)
	if len(words) > 2 {
		return strings.Join(words[:2], " ")
	}
	return topic
}

func removeStrayCommas(topic string) string {
	return strings.TrimSpace(strings.TrimLeft(strings.TrimRight(topic, ","), ","))
}

func isValidTopic(topic string) bool {
	genericPhrases := []string{"and", "avoiding", "based on", "content", "for", "here", "keeping", "with"}
	topicLower := strings.ToLower(topic)
	for _, phrase := range genericPhrases {
		if strings.Contains(topicLower, phrase) {
			return false
		}
	}
	return len(topic) > 1
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
