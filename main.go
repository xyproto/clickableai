package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/xyproto/ollamaclient/v2"
)

const mainPrompt = "Generate correct, interesting and technical documentation about these keywords, in Markdown: "

type PageData struct {
	Keywords       []string
	MarkdownOutput template.HTML
}

var (
	currentKeywords = []string{"Assembly", "C", "Go", "Rust", "Python", "Concurrency", "WebAssembly", "JavaScript", "AI", "Machine Learning"}
	keywordTrail    = []string{}
)

func main() {
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
			fields[i] = betweenQuotes(field)
		}
		if len(fields) > 1 {
			newKeywords = fields
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
