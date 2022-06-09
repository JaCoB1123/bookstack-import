package main

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

type bookStackClient struct {
	*resty.Client
}

type httpErrorResponse struct {
	Error httpError `json:"error"`
}
type httpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (err httpError) Error() string {
	return "HTTP Error"
}

func NewBookStackClient(path, tokenID, tokenSecret string) (*bookStackClient, error) {
	if path == "" || tokenID == "" || tokenSecret == "" {
		return nil, fmt.Errorf("URL, Token-ID and Token-Secret have to be specified")
	}

	client := resty.New()
	client.SetError(httpErrorResponse{})
	client.SetBaseURL(path)
	client.SetHeader("Authorization", "Token "+tokenID+":"+tokenSecret)

	_, err := client.R().Get("/api/docs.json")
	if err != nil {
		return nil, fmt.Errorf("get list of endpoints: %w", err)
	}

	return &bookStackClient{client}, nil
}
