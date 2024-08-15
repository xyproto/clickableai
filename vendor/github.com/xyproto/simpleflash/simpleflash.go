package simpleflash

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/vertexai/genai"
	"github.com/allegro/bigcache/v3"
	"github.com/xyproto/env"
	"github.com/xyproto/multimodal"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

type SimpleFlash struct {
	ModelName           string
	MultiModalModelName string
	ProjectLocation     string
	ProjectID           string
	Client              *genai.Client
	Cache               *bigcache.BigCache
	Timeout             time.Duration
}

func New(modelName, multiModalModelName, projectLocation, projectID string, cache bool) (*SimpleFlash, error) {
	sf := &SimpleFlash{
		ModelName:           env.Str("MODEL_NAME", modelName),
		MultiModalModelName: env.Str("MULTI_MODAL_MODEL_NAME", multiModalModelName),
		ProjectLocation:     env.Str("PROJECT_LOCATION", projectLocation),
		ProjectID:           env.Str("PROJECT_ID", projectID),
		Timeout:             3 * time.Minute,
	}

	// Initialize the genai client
	ctx := context.Background()
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("Failed to obtain default credentials: %v", err)
	}
	genaiClient, err := genai.NewClient(ctx, sf.ProjectID, sf.ProjectLocation, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %v", err)
	}
	sf.Client = genaiClient

	// Initialize cache if the cache parameter is true
	if cache {
		err := sf.InitCache()
		if err != nil {
			return nil, fmt.Errorf("Failed to initialize cache: %v", err)
		}
	}

	return sf, nil
}

// InitCache initializes the BigCache cache
func (sf *SimpleFlash) InitCache() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config := bigcache.DefaultConfig(24 * time.Hour)
	config.HardMaxCacheSize = 256 // MB
	config.StatsEnabled = false
	config.Verbose = false

	cache, err := bigcache.New(ctx, config)
	if err != nil {
		return err
	}
	sf.Cache = cache
	return nil
}

// QueryGemini processes a prompt with optional temperature, base64-encoded data, and MIME type for the data.
func (sf *SimpleFlash) QueryGemini(prompt string, temperature *float64, base64Data, dataMimeType *string) (string, error) {
	modelName := sf.ModelName

	// Generate a unique cache key based on prompt and optionally on temperature, base64Data, and dataMimeType
	cacheKeyComponents := prompt
	if temperature != nil {
		cacheKeyComponents += fmt.Sprintf("%f", *temperature)
	}
	if base64Data != nil {
		cacheKeyComponents += *base64Data
		modelName = sf.MultiModalModelName
	}
	if dataMimeType != nil {
		cacheKeyComponents += *dataMimeType
	}
	cacheKey := fmt.Sprintf("%x", sha256.Sum256([]byte(cacheKeyComponents)))

	// Check cache for existing entry
	if sf.Cache != nil {
		if entry, err := sf.Cache.Get(cacheKey); err == nil {
			return string(entry), nil
		}
	}

	// Initialize the multimodal instance, using the provided temperature if available
	mm := multimodal.New(modelName, 0.0) // Default temperature
	mm.SetTimeout(sf.Timeout)
	if temperature != nil {
		mm = multimodal.New(modelName, float32(*temperature))
	}
	mm.AddText(prompt)

	// If base64Data and dataMimeType are provided, decode the data and add it to the multimodal instance
	if base64Data != nil && dataMimeType != nil {
		data, err := base64.StdEncoding.DecodeString(*base64Data)
		if err != nil {
			return "", fmt.Errorf("failed to decode base64 data: %v", err)
		}
		mm.AddData(*dataMimeType, data)
	}

	ctx, cancel := context.WithTimeout(context.Background(), sf.Timeout)
	defer cancel()

	// Submit the multimodal query and process the result
	result, err := mm.SubmitToClient(ctx, sf.Client)
	if err != nil {
		return "", fmt.Errorf("failed to process response: %v", err)
	}

	result = strings.TrimSpace(result)

	// Store the new result in the cache
	if sf.Cache != nil {
		_ = sf.Cache.Set(cacheKey, []byte(result))
	}

	return result, nil
}

// CountTextTokens tries to count the number of tokens in the given prompt, using the VertexAI API
func (sf *SimpleFlash) CountTextTokens(prompt string) (int, error) {
	mm := multimodal.New(sf.ModelName, 0.0)
	mm.SetTimeout(sf.Timeout)

	ctx, cancel := context.WithTimeout(context.Background(), sf.Timeout)
	defer cancel()

	return mm.CountTextTokensWithClient(ctx, sf.Client, prompt)
}
