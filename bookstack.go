package main

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

type bookStackClient struct {
	*resty.Client
}

func NewBookStackClient(url, tokenID, tokenSecret string) (*bookStackClient, error) {
	if url == "" || tokenID == "" || tokenSecret == "" {
		return nil, fmt.Errorf("URL, Token-ID and Token-Secret have to be specified")
	}

	client := resty.New()
	client.SetBaseURL(url)
	client.SetHeader("Authorization", "Token "+tokenID+":"+tokenSecret)

	_, err := client.R().Get("/api/docs.json")
	if err != nil {
		return nil, fmt.Errorf("get list of endpoints: %w", err)
	}

	return &bookStackClient{client}, nil
}
