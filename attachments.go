package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

type attachment struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Extension  string `json:"extension"`
	UploadedTo int    `json:"uploaded_to"`
}

func (client bookStackClient) UploadAttachment(pageID int, name string, path string) (*attachment, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer fd.Close()

	// TODO check for existing attachment by hash

	fileName := filepath.Base(fd.Name())

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	filepart, err := mw.CreateFormFile("file", fd.Name())
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}

	_, err = io.Copy(filepart, fd)
	if err != nil {
		return nil, fmt.Errorf("copy file data: %w", err)
	}

	uploadedTo, err := mw.CreateFormField("uploaded_to")
	if err != nil {
		return nil, fmt.Errorf("create field uploaded_to: %w", err)
	}

	fmt.Fprintf(uploadedTo, "%d", pageID)

	nameField, err := mw.CreateFormField("name")
	if err != nil {
		return nil, fmt.Errorf("create field name: %w", err)
	}

	fmt.Fprintf(nameField, "%s", fileName)

	err = mw.Close()
	if err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	resp, err := client.R().
		SetBody(&buf).
		SetHeader("Content-Type", mw.FormDataContentType()).
		SetResult(attachment{}).
		Post("/api/attachments")
	if err != nil || resp.StatusCode() > 399 {
		return nil, fmt.Errorf("http POST: %v: %w", resp, err)
	}
	return resp.Result().(*attachment), nil
}
