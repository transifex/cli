package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type ResourceTranslationsDownloads struct {
	Attributes struct {
		Content         *[]byte `json:"content,omitempty"`
		ContentEncoding *string `json:"content_encoding,omitempty"`
		FileType        *string `json:"file_type,omitempty"`
		Mode            *string `json:"mode,omitempty"`
		Pseudo          *bool   `json:"pseudo,omitempty"`
		DateCreated     *string `json:"date_created,omitempty"`
		DateModified    *string `json:"date_modified,omitempty"`
		Errors          *[]struct {
			Code   *string `json:"code,omitempty"`
			Detail *string `json:"detail,omitempty"`
		} `json:"errors,omitempty"`
		Status *string `json:"status,omitempty"`
	} `json:"attributes"`
	ID    *string `json:"id,omitempty"`
	Links *struct {
		Self string `json:"self"`
	} `json:"links,omitempty"`
	Relationships struct {
		Language struct {
			Data struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"data"`
			Links *struct {
				Related *string `json:"related,omitempty"`
			} `json:"links,omitempty"`
		} `json:"language,omitempty"`
		Resource struct {
			Data struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"data"`
			Links *struct {
				Related *string `json:"related,omitempty"`
			} `json:"links,omitempty"`
		} `json:"resource,omitempty"`
	} `json:"relationships,omitempty"`
	Type string `json:"type"`
}

func (c *Client) getResourceTranslationsDownload(downloadID string) (*ResourceTranslationsDownloads, error) {
	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/resource_translations_async_downloads/%s", c.Host, downloadID),
		nil,
	)
	translationsDownload := ResourceTranslationsDownloads{}
	err := c.DoRequest(req, &translationsDownload)

	if err != nil {
		var e *ExternalRedirectError

		if errors.As(err, &e) {
			req, _ := http.NewRequest(
				"GET",
				e.URL.String(),
				nil,
			)
			response, err := c.HTTPClient.Do(req)
			bodyBytes, err := ioutil.ReadAll(response.Body)
			if err != nil {
				return nil, err
			}
			translationsDownload.Attributes.Content = &bodyBytes
			return &translationsDownload, nil
		}
		return nil, err
	}
	return &translationsDownload, nil
}

// PollStatus will keep 'pinging' the status API until timeout is reached or status returned is either successful or in error.
func (c *Client) pollResourceTranslationsDownload(ctx context.Context, downloadID string, pollInterval time.Duration) (*ResourceTranslationsDownloads, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	failedStatus := "failed"
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case <-ticker.C:
			translationsDownload, err := c.getResourceTranslationsDownload(downloadID)
			if err != nil {
				return nil, err
			}
			status := translationsDownload.Attributes.Status
			if status != nil && *status == failedStatus {
				if err = PrintResponse(translationsDownload.Attributes.Errors); err != nil {
					return translationsDownload, nil
				}
				return translationsDownload, nil
			}
			if translationsDownload.Attributes.Content != nil {
				return translationsDownload, nil
			}
		}
	}
}

func (c *Client) createResourceTranslationsDownload(resourceID string, languageID string, mode string, xliff bool) (*ResourceTranslationsDownloads, error) {
	translationsDownload := ResourceTranslationsDownloads{}
	translationsDownload.Type = "resource_translations_async_downloads"
	translationsDownload.Attributes.Mode = &mode
	if xliff {
		xliffFileType := "xliff"
		translationsDownload.Attributes.FileType = &xliffFileType
	}
	translationsDownload.Relationships.Resource.Data.Type = "resources"
	translationsDownload.Relationships.Resource.Data.ID = resourceID
	translationsDownload.Relationships.Language.Data.Type = "languages"
	translationsDownload.Relationships.Language.Data.ID = languageID

	body, err := JSONMarshal(map[string]ResourceTranslationsDownloads{
		"data": translationsDownload,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/resource_translations_async_downloads", c.Host),
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}
	if err := c.DoRequest(req, &translationsDownload); err != nil {
		return nil, err
	}
	return &translationsDownload, nil
}
