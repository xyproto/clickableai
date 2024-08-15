package main

import (
	"crypto/md5"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/simpleflash"
)

const (
	textModel       = "gemini-1.5-flash-001"
	multiModalModel = "gemini-1.0-pro-vision-001"
	mainPrompt      = "Generate correct, interesting and technical documentation about these keywords, in Markdown: "
)

type PageData struct {
	Keywords        []string
	AvailableTopics []string
	MarkdownOutput  template.HTML
}

var (
	projectLocation = env.Str("PROJECT_LOCATION", "europe-west4") // europe-west4 is the default
	projectID       = env.Str("PROJECT_ID")
	sf              *simpleflash.SimpleFlash
	initialTopics   = []string{"Assembly", "C", "Go", "Rust", "Python", "Concurrency", "WebAssembly", "JavaScript", "AI", "Machine Learning"}
	sessionMap      = make(map[string][]string)
)

func main() {
	// Check if PROJECT_ID is set
	if projectID == "" {
		fmt.Fprintln(os.Stderr, "Error: PROJECT_ID environment variable is not set.")
		return
	}

	// Initialize the SimpleFlash instance
	var err error
	sf, err = simpleflash.New(textModel, multiModalModel, projectLocation, projectID, true)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return
	}

	http.HandleFunc("/", handler)
	http.HandleFunc("/generate", generateHandler)
	http.HandleFunc("/clear", clearHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// generateFingerprint creates a unique identifier based on the request headers and IP address
func generateFingerprint(r *http.Request) string {
	h := md5.New()

	// Concatenate various request headers and the IP address
	io.WriteString(h, r.RemoteAddr)
	io.WriteString(h, r.Header.Get("User-Agent"))
	io.WriteString(h, r.Header.Get("Accept-Language"))
	io.WriteString(h, r.Header.Get("Accept-Encoding"))

	// You can add more headers if needed to increase uniqueness
	return fmt.Sprintf("%x", h.Sum(nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	sessionID := generateFingerprint(r)

	keywords, ok := sessionMap[sessionID]
	if !ok {
		keywords = []string{} // Start with an empty personal list of keywords
		sessionMap[sessionID] = keywords
	}

	// Only generate content if a keyword is selected
	var markdown string
	if len(keywords) > 0 {
		markdown, _ = generateMarkdownAndKeywords(keywords)
	}

	data := PageData{
		Keywords:        keywords,
		AvailableTopics: initialTopics,
		MarkdownOutput:  template.HTML(markdown),
	}

	tmpl := template.Must(template.ParseFiles("index.html"))
	tmpl.Execute(w, data)
}

func generateHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := generateFingerprint(r)

	keywords, ok := sessionMap[sessionID]
	if !ok {
		keywords = []string{} // Start with an empty personal list of keywords
	}

	keyword := r.URL.Query().Get("keyword")
	if keyword != "" {
		keywords = append(keywords, keyword)
		sessionMap[sessionID] = keywords
	}

	markdown, _ := generateMarkdownAndKeywords(keywords)

	data := PageData{
		Keywords:        keywords,
		AvailableTopics: initialTopics,
		MarkdownOutput:  template.HTML(markdown),
	}

	tmpl := template.Must(template.ParseFiles("index.html"))
	tmpl.Execute(w, data)
}

func clearHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := generateFingerprint(r)
	sessionMap[sessionID] = []string{} // Clear the personal list of keywords
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func generateMarkdownAndKeywords(trail []string) (string, []string) {
	prompt := mainPrompt + strings.Join(trail, " -> ")

	temperature := 0.0
	output, err := sf.QueryGemini(prompt, &temperature, nil, nil)
	if err != nil {
		fmt.Println("Error:", err)
		return "Error: Could not generate output", nil
	}

	// The generation of new keywords is omitted since it's based on the trail the user builds up

	return output, nil
}
