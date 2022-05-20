package main

import "fmt"

type booksResponse struct {
	Data []book `json:"data"`
}

type book struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

func (client bookStackClient) GetBooks() (*booksResponse, error) {
	booksResp, err := client.R().
		SetResult(booksResponse{}).
		Get("/api/books")
	if err != nil {
		return nil, fmt.Errorf("get of books: %w", err)
	}

	return booksResp.Result().(*booksResponse), nil
}

func (client bookStackClient) CreateBook(name string) (*book, error) {
	resp, err := client.R().
		SetBody(book{Name: name}).
		SetResult(book{}).
		Post("/api/books")
	if err != nil {
		return nil, fmt.Errorf("create book: %w", err)
	}
	return resp.Result().(*book), nil
}
