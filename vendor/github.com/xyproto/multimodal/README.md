# multimodal

Package for making calls to multimodal prompts in Google Cloud easier to deal with.

## Example use

```go
package main

import (
    "fmt"
    "log"

    "github.com/xyproto/multimodal"
)

func main() {
    // Select the model and temperature
    mm := multimodal.New("gemini-1.0-pro-vision", 0.4)

    // Build a prompt
    mm.AddImage("frog.png")
    mm.AddURI("gs://generativeai-downloads/images/scones.jpg")
    mm.AddText("describe what is common for these two images")

    // Use your location and project ID for a multimodal Vertex AI model in Google Cloud
    const location = "us-central1"
    const projectID = "123412341234"

    // Submit the prompt and get a reponse
    response, err := mm.Submit(projectID, location)
    if err != nil {
        log.Fatalln(err)
    }

    fmt.Println(response)
}
```

## General info

* Version: 1.3.3
* License: Apache2
