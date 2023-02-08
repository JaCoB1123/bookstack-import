package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
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
