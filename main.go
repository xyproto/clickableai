package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/xyproto/ollamaclient/v2"
)

type PageData struct {
	Keywords       []string
	MarkdownOutput template.HTML
}

var currentKeywords = []string{"Assembly", "C", "Go", "Rust"}
var keywordTrail = []string{}

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
		MarkdownOutput: template.HTML(markdown), // Safely render HTML content
	}
	currentKeywords = newKeywords

	tmpl := template.Must(template.ParseFiles("index.html"))
	tmpl.Execute(w, data)
}

func generateMarkdownAndKeywords(trail []string) (string, []string) {
	prompt := "Generate a Markdown document about: " + fmt.Sprint(trail)

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

	// Simulate new keyword generation based on output
	newKeywords := []string{"Python", "Concurrency", "WebAssembly"}

	return output, newKeywords
}
