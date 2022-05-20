package main

import "fmt"

type chaptersResponse struct {
	Data []chapter `json:"data"`
}

type chapter struct {
	ID          int    `json:"id"`
	BookID      int    `json:"book_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

func (client bookStackClient) GetChapters() (*chaptersResponse, error) {
	resp, err := client.R().
		SetResult(chaptersResponse{}).
		Get("/api/chapters")
	if err != nil {
		return nil, fmt.Errorf("get of chapters: %w", err)
	}

	return resp.Result().(*chaptersResponse), nil
}

func (client bookStackClient) CreateChapter(bookID int, name string) (*chapter, error) {
	resp, err := client.R().
		SetBody(chapter{
			BookID: bookID,
			Name:   name}).
		SetResult(chapter{}).
		Post("/api/chapters")
	if err != nil {
		return nil, fmt.Errorf("create chapter: %w", err)
	}
	return resp.Result().(*chapter), nil
}
