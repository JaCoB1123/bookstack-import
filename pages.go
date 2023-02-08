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
	if err != nil || resp.StatusCode() > 399 {
		return nil, fmt.Errorf("get of pagers: %s", resp)
	}

	return resp.Result().(*pagesResponse), nil
}

func (client bookStackClient) CreatePage(chapterID int, name string, content []byte) (*page, error) {
	resp, err := client.R().
		SetBody(page{
			ChapterID: chapterID,
			Name:      name,
			Markdown:  string(content)}).
		SetResult(page{}).
		Post("/api/pages")
	if err != nil || resp.StatusCode() > 399 {
		return nil, fmt.Errorf("create page: %s", resp)
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
	if err != nil || resp.StatusCode() > 399 {
		return nil, fmt.Errorf("update page: %s", resp)
	}
	return resp.Result().(*page), nil
}
