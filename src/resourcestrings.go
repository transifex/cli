package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type ResourceStringsUpload struct {
	Attributes struct {
		DateCreated  *string `json:"date_created,omitempty"`
		DateModified *string `json:"date_modified,omitempty"`
		Details      *struct {
			StringsCreated *int `json:"strings_created,omitempty"`
			StringsDeleted *int `json:"strings_deleted,omitempty"`
			StringsSkipped *int `json:"strings_skipped,omitempty"`
			StringsUpdated *int `json:"strings_updated,omitempty"`
		} `json:"details,omitempty"`
		Errors *[]struct {
			Code   *string `json:"code,omitempty"`
			Detail *string `json:"detail,omitempty"`
		} `json:"errors,omitempty"`
		Status *string `json:"status,omitempty"`
	} `json:"attributes"`
	ID    string `json:"id"`
	Links *struct {
		Self *string `json:"self,omitempty"`
	} `json:"links,omitempty"`
	Relationships struct {
		Resource struct {
			Data struct {
				ID   string `json:"id,omitempty"`
				Type string `json:"type,omitempty"`
			} `json:"data"`
			Links struct {
				Related string `json:"related,omitempty"`
			} `json:"links,omitempty"`
		} `json:"resource,omitempty"`
	} `json:"relationships,omitempty"`
	Type string `json:"type"`
}

func (c *Client) getResourceStringsUpload(resourceStringsUploadID string) (*ResourceStringsUpload, error) {
	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/resource_strings_async_uploads/%s", c.Host, resourceStringsUploadID),
		nil,
	)
	resourceStringsUpload := ResourceStringsUpload{}
	if err := c.DoRequest(req, &resourceStringsUpload); err != nil {
		return nil, err
	}
	return &resourceStringsUpload, nil
}

func (c *Client) pollResourceStringsUpload(ctx context.Context, uploadID string, pollInterval time.Duration) (*ResourceStringsUpload, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	failedStatus := "failed"
	successStatus := "succeeded"

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			sourceUpload, err := c.getResourceStringsUpload(uploadID)
			if err != nil {
				return nil, err
			}
			status := sourceUpload.Attributes.Status
			if status != nil && *status == failedStatus {
				if err = PrintResponse(sourceUpload.Attributes.Errors); err != nil {
					return sourceUpload, nil
				}
				return sourceUpload, nil
			}
			if status != nil && *status == successStatus {
				return sourceUpload, nil
			}
		}
	}
}

func (c *Client) createResourceStringsUpload(resourceID string, filePath string) (*ResourceStringsUpload, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	resourceField, err := writer.CreateFormField("resource")
	if err != nil {
		return nil, err
	}
	resourceField.Write([]byte(resourceID))

	contentField, err := writer.CreateFormFile("content", filepath.Base(file.Name()))
	if err != nil {
		return nil, err
	}
	io.Copy(contentField, file)
	writer.Close()

	req, _ := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/resource_strings_async_uploads", c.Host),
		body,
	)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resourceStringsUpload := ResourceStringsUpload{}
	if err := c.DoRequest(req, &resourceStringsUpload); err != nil {
		return nil, err
	}
	return &resourceStringsUpload, nil
}
