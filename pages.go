package main

import "fmt"

type pagesResponse struct {
	Data []page `json:"data"`
}

type page struct {
	ID          int    `json:"id"`
	BookID      int    `json:"book_id"`
	ChapterID   int    `json:"chapter_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

func (client bookStackClient) GetPages() (*pagesResponse, error) {
	resp, err := client.R().
		SetResult(pagesResponse{}).
		Get("/api/pages")
	if err != nil {
		return nil, fmt.Errorf("get of pagers: %w", err)
	}

	return resp.Result().(*pagesResponse), nil
}

func (client bookStackClient) CreatePage(bookID, chapterID int, name string) (*page, error) {
	resp, err := client.R().
		SetBody(page{
			BookID:    bookID,
			ChapterID: chapterID,
			Name:      name}).
		SetResult(page{}).
		Post("/api/pages")
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}
	return resp.Result().(*page), nil
}
