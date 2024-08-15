package main

import (
	"fmt"
	"html/template"
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
	Keywords       []string
	MarkdownOutput template.HTML
}

var (
	projectLocation = env.Str("PROJECT_LOCATION", "europe-west4") // europe-west4 is the default
	projectID       = env.Str("PROJECT_ID")
	sf              *simpleflash.SimpleFlash
	currentKeywords = []string{"Assembly", "C", "Go", "Rust", "Python", "Concurrency", "WebAssembly", "JavaScript", "AI", "Machine Learning"}
	keywordTrail    = []string{}
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
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("index.html"))
	data := PageData{
		Keywords:       currentKeywords,
		MarkdownOutput: "",
	}
	tmpl.Execute(w, data)
}

func generateHandler(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("keyword")
	if keyword != "" {
		keywordTrail = append(keywordTrail, keyword)
	}

	markdown, newKeywords := generateMarkdownAndKeywords(keywordTrail)

	data := PageData{
		Keywords:       newKeywords,
		MarkdownOutput: template.HTML(markdown),
	}
	currentKeywords = newKeywords

	tmpl := template.Must(template.ParseFiles("index.html"))
	tmpl.Execute(w, data)
}

func generateMarkdownAndKeywords(trail []string) (string, []string) {
	prompt := mainPrompt + strings.Join(trail, " -> ")

	temperature := 0.0
	output, err := sf.QueryGemini(prompt, &temperature, nil, nil)
	if err != nil {
		fmt.Println("Error:", err)
		return "Error: Could not generate output", nil
	}

	newKeywords := []string{"Networking", "Databases", "Kubernetes"}
	followUpPrompt := "Generate 10 interesting follow-up keywords that relate to the following text:\n" + output + "\n\nOnly output the slice of strings, as Go code. No commentary!"
	followUpKeywordsString, err := sf.QueryGemini(followUpPrompt, &temperature, nil, nil)
	if err == nil {
		followUpKeywordsString = strings.TrimPrefix(followUpKeywordsString, "```go")
		followUpKeywordsString = strings.TrimPrefix(followUpKeywordsString, "```")
		followUpKeywordsString = strings.TrimSuffix(followUpKeywordsString, "```")
		fields := strings.Split(followUpKeywordsString, ",")
		for i, field := range fields {
			fields[i] = betweenQuotes(field)
		}
		if len(fields) > 1 {
			newKeywords = make([]string, 0, len(fields))
			for _, field := range fields {
				nkw := strings.TrimSpace(field)
				if len(nkw) > 2 {
					if strings.Contains(nkw, " ") {
						newKeywords = append(newKeywords, nkw)
					} else {
						newKeywords = append(newKeywords, capitalize(nkw))
					}
				}
			}
		}
	}

	return output, newKeywords
}

// capitalize the first character of the string and make all other characters lowercase
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

// betweenQuotes can return what's between the double quotes in the given string.
// If fewer than two double quotes are found, return the original string.
func betweenQuotes(orig string) string {
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
