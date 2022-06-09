package main

import (
	"fmt"
	"strconv"
)

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
	Markdown    string `json:"markdown"`
}

func (p page) String() string {
	return strconv.Itoa(p.ChapterID) + ": " + p.Name
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

func (client bookStackClient) CreatePage(bookID, chapterID int, name string, content []byte) (*page, error) {
	resp, err := client.R().
		SetBody(page{
			BookID:    bookID,
			ChapterID: chapterID,
			Name:      name,
			Markdown:  string(content)}).
		SetResult(page{}).
		Post("/api/pages")
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}
	return resp.Result().(*page), nil
}

type pageContentRequest struct {
	Markdown string `json:"markdown"`
}

func (client bookStackClient) UpdatePageContent(pageID int, content []byte) (*page, error) {
	resp, err := client.R().
		SetBody(pageContentRequest{
			Markdown: string(content)}).
		SetResult(page{}).
		Put("/api/pages/" + strconv.Itoa(pageID))
	if err != nil {
		return nil, fmt.Errorf("update page: %w", err)
	}
	return resp.Result().(*page), nil
}
