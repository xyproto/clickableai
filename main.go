package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/simpleflash"
)

const (
	textModel           = "gemini-1.5-flash-001"
	multiModalModel     = "gemini-1.0-pro-vision-001"
	mainPrompt          = "Generate a concise, technical Markdown document based on these keywords. Avoid commentary: "
	topicPrompt         = "Generate exactly 10 concise topics based on these keywords and the following content. Each topic should be 1-2 words max. Avoid redundancy: "
	generalTopicPrompt  = "Generate 10 general keywords based on the following Markdown content. Each keyword should be 1-2 words max. Avoid redundancy: "
)

type PageData struct {
	Keywords        []string
	MarkdownOutput  string
	AvailableTopics []string
}

var (
	projectLocation = env.Str("PROJECT_LOCATION", "europe-west4")
	projectID       = env.Str("PROJECT_ID")
	sf              *simpleflash.SimpleFlash
	initialTopics   = []string{"Assembly", "C", "Go", "Rust", "Python", "Concurrency", "WebAssembly", "JavaScript", "AI", "Machine Learning"}
)

func main() {
	if projectID == "" {
		fmt.Fprintln(os.Stderr, "Error: PROJECT_ID environment variable is not set.")
		return
	}

	var err error
	sf, err = simpleflash.New(textModel, multiModalModel, projectLocation, projectID, true)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return
	}

	http.HandleFunc("/", handler)
	http.HandleFunc("/generate", generateHandler)
	http.HandleFunc("/generate_topics", generateTopicsHandler)

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Keywords:        []string{},
		MarkdownOutput:  "",
		AvailableTopics: initialTopics,
	}

	tmpl := template.Must(template.ParseFiles("index.html"))
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing template: %s\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

	fmt.Printf("Generating new topics for %v\n", keywords)

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

	//log.Printf("Sending prompt to Gemini for Markdown generation: %s\n", prompt)

	temperature := 0.0
	output, err := sf.QueryGemini(prompt, &temperature, nil, nil)
	if err != nil {
		fmt.Println("Error:", err)
		return "Error: Could not generate output", nil
	}

	return output, nil
}

func generateNewTopics(keywords []string, markdown string) []string {
	fmt.Printf("Generating new topics for %v\n", keywords)

	prompt := topicPrompt + strings.Join(keywords, ", ") + " | Content: " + markdown

	//log.Printf("Sending prompt to Gemini for topic generation: %s\n", prompt)

	temperature := 0.5
	topicsOutput, err := sf.QueryGemini(prompt, &temperature, nil, nil)
	if err != nil {
		fmt.Println("Error:", err)
		return []string{"Error: Could not generate topics"}
	}

	topics := extractAndShortenTopics(topicsOutput, keywords)

	return topics
}

func generateGeneralTopics(markdown string) []string {
	prompt := generalTopicPrompt + markdown

	fmt.Printf("Generating general new topics for %d bytes of Markdown.\n", len(markdown))

	//log.Printf("Sending general prompt to Gemini for topic generation: %s\n", prompt)

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
	re := regexp.MustCompile(`([a-zA-Z0-9\-\_ ]+,)+[a-zA-Z0-9\-\_ ]+`)
	match := re.FindString(output)

	if match == "" {
		log.Println("No valid comma-separated list found in the output")
		return []string{"Error: No valid topics found"}
	}

	topics := strings.Split(match, ",")

	// Shorten each topic, especially if it's redundant with existing keywords
	for i, topic := range topics {
		topic = strings.TrimSpace(topic)
		for _, keyword := range keywords {
			if strings.Contains(strings.ToLower(topic), strings.ToLower(keyword)) {
				topic = strings.Replace(topic, keyword, "", -1)
				topic = strings.TrimSpace(topic)
			}
		}
		topics[i] = shortenToTwoWords(topic)
	}

	if len(topics) > 10 {
		topics = topics[:10]
	}

	return topics
}

// shortenToTwoWords shortens a string to a maximum of two words.
func shortenToTwoWords(topic string) string {
	words := strings.Fields(topic)
	if len(words) > 2 {
		return strings.Join(words[:2], " ")
	}
	return topic
}
