package clickableai

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"
)

var tmpl *template.Template

// PageData holds the data to be rendered in the HTML template
type PageData struct {
	Keywords       []string
	MarkdownOutput template.HTML
	ExtraInHead    template.HTML
}

// InitTemplate initializes the template with the provided HTML content
func InitTemplate(indexHTML string) {
	tmpl = template.Must(template.New("index").Parse(indexHTML))
}

// Handler handles the main page rendering with provided keywords and markdown content
func Handler(w http.ResponseWriter, r *http.Request, keywords []string, markdown string, extraInHead string) {
	data := PageData{
		Keywords:       keywords,
		MarkdownOutput: template.HTML(markdown),
		ExtraInHead:    template.HTML(extraInHead),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Printf("Error executing template: %s\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
	w.Write(buf.Bytes())
}

// RobotsHandler serves the robots.txt file
func RobotsHandler(w http.ResponseWriter, r *http.Request, robots string) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(robots))
}

// GithubLogoHandler serves the embedded GitHub logo image
func GithubLogoHandler(w http.ResponseWriter, r *http.Request, githublogo []byte) {
	w.Header().Set("Content-Type", "image/png")
	w.Write(githublogo)
}

// MarkdownJSHandler serves the embedded markdown-it.min.js file
func MarkdownJSHandler(w http.ResponseWriter, r *http.Request, markdownJS []byte) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Write(markdownJS)
}

// ExtractAndShortenTopics processes the output to remove redundant phrases and shorten topics to 1-2 words.
func ExtractAndShortenTopics(output string, keywords []string) []string {
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
			topics = append(topics, strings.Trim(topic, "\""))
		}
	}

	if len(topics) > 10 {
		topics = topics[:10]
	}

	return topics
}

// Utility functions for processing topics

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

// BetweenQuotes returns the substring between double quotes in the given string.
// If fewer than two double quotes are found, returns the original string.
func BetweenQuotes(orig string) string {
	const q = `"`
	if strings.Count(orig, q) >= 2 {
		posa := strings.Index(orig, q) + 1
		posb := strings.LastIndex(orig, q)
		if posa >= posb {
			return orig
		}
		return orig[posa:posb]
	}
	return orig
}
