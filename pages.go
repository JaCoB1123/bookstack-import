package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
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

type attachment struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Extension  string `json:"extension"`
	UploadedTo int    `json:"uploaded_to"`
}

func (client bookStackClient) UploadAttachment(pageID int, name string, path string) (*attachment, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file to upload: %w", err)
	}
	defer fd.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			"file", fd.Name()))
	h.Set("Content-Type", "image/png")
	filepart, err := mw.CreatePart(h)

	//filepart, err := mw.CreateFormFile("file", fd.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to create new form part: %w", err)
	}

	_, err = io.Copy(filepart, fd)
	if err != nil {
		return nil, fmt.Errorf("failed to write form part: %w", err)
	}

	uploadedTo, err := mw.CreateFormField("uploaded_to")
	if err != nil {
		return nil, fmt.Errorf("failed to create new form part: %w", err)
	}

	fmt.Fprintf(uploadedTo, "%d", pageID)

	nameField, err := mw.CreateFormField("name")
	if err != nil {
		return nil, fmt.Errorf("failed to create new form part: %w", err)
	}

	fmt.Fprintf(nameField, "%s", fd.Name())

	err = mw.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to prepare form: %w", err)
	}

	resp, err := client.R().
		SetBody(&buf).
		SetHeader("Content-Type", mw.FormDataContentType()).
		SetResult(attachment{}).
		Post("/api/attachments")
	if err != nil || resp.StatusCode() > 399 {
		return nil, fmt.Errorf("create attachment: %s", resp)
	}
	return resp.Result().(*attachment), nil
}
