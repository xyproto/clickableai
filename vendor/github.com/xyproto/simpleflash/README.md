# SimpleFlash

A simple way to use Gemini and the gemini-1.5-flash model (or other Gemini-models).

Example use:

```go
package main

import (
    "fmt"
    "os"
    "time"

    "github.com/xyproto/env/v2"
    "github.com/xyproto/simpleflash"
)

func main() {
    const (
        textModel       = "gemini-1.5-flash-001"
        multiModalModel = "gemini-1.0-pro-vision-001"
    )

    var (
        projectLocation = env.Str("PROJECT_LOCATION", "europe-west4") // europe-west4 is just the default
        projectID       = env.Str("PROJECT_ID")
    )

    if projectID == "" {
        fmt.Fprintln(os.Stderr, "Error: PROJECT_ID environment variable is not set.")
        return
    }

    sf, err := simpleflash.New(textModel, multiModalModel, projectLocation, projectID, true)
    if err != nil {
        fmt.Fprintln(os.Stderr, "Error:", err)
        return
    }

    sf.Timeout = 10 * time.Second

    const prompt = "Write a haiku about the color of cows."

    // Query Gemini with the prompt, nothing multimodal
    output, err := sf.QueryGemini(prompt, nil, nil, nil)
    if err != nil {
        fmt.Fprintln(os.Stderr, "Error:", err)
        return
    }

    fmt.Println(output)
}
```

Building and running the example (in `cmd/simple`):

```sh
cd cmd/simple

# Use your own Google Cloud Project ID here, and also log in with `gcloud auth application-default login` **or** deploy and run it within Google Cloud:
export PROJECT_ID=44444444444

# Build the executable
go build

# Run the program and observe the output
./simple
Black and white patches,
Sun-kissed brown, a gentle breeze,
Grazing in the field.
```

### General info

* Version: 1.0.1
* License: Apache 2
