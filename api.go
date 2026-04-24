// Beebeeb rclone backend — HTTP client for the Beebeeb API
// Copyright (C) 2026 Beebeeb
// SPDX-License-Identifier: AGPL-3.0-or-later

package beebeeb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// FileEntry represents a single file or folder returned by the API.
type FileEntry struct {
	ID            string    `json:"id"`
	NameEncrypted string    `json:"name_encrypted"`
	MimeType      string    `json:"mime_type"`
	SizeBytes     int64     `json:"size_bytes"`
	IsFolder      bool      `json:"is_folder"`
	ChunkCount    int       `json:"chunk_count"`
	ParentID      *string   `json:"parent_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Client is an HTTP client for the Beebeeb API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new API client from the given Config.
func NewClient(cfg *Config) *Client {
	return &Client{
		baseURL: strings.TrimRight(cfg.APIURL, "/"),
		token:   cfg.Token,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// do executes an HTTP request, adding the Authorization header.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	return c.httpClient.Do(req)
}

// ListFiles returns the files and folders under the given parentID.
// Pass an empty string for parentID to list the root.
func (c *Client) ListFiles(parentID string) ([]FileEntry, error) {
	u := c.baseURL + "/api/v1/files"
	if parentID != "" {
		u += "?parent_id=" + url.QueryEscape(parentID)
	}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, readAPIError(resp)
	}

	var result struct {
		Files []FileEntry `json:"files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("list files: decode: %w", err)
	}
	return result.Files, nil
}

// UploadFile uploads the contents of r as a file with the given name under
// parentID. Pass empty string for parentID to upload to the root.
func (c *Client) UploadFile(name string, parentID string, sizeBytes int64, mimeType string, r io.Reader) (*FileEntry, error) {
	// Build multipart body: metadata part + chunk_0 part.
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// Metadata part.
	meta := map[string]interface{}{
		"name_encrypted": name,
		"size_bytes":     sizeBytes,
	}
	if parentID != "" {
		meta["parent_id"] = parentID
	}
	if mimeType != "" {
		meta["mime_type"] = mimeType
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("upload: marshal metadata: %w", err)
	}

	metaPart, err := w.CreateFormField("metadata")
	if err != nil {
		return nil, fmt.Errorf("upload: create metadata field: %w", err)
	}
	if _, err := metaPart.Write(metaJSON); err != nil {
		return nil, fmt.Errorf("upload: write metadata: %w", err)
	}

	// Single chunk part (chunk_0). For large files a chunked uploader
	// would be needed; this implementation sends the file as one chunk.
	chunkPart, err := w.CreateFormFile("chunk_0", "chunk_0")
	if err != nil {
		return nil, fmt.Errorf("upload: create chunk field: %w", err)
	}
	if _, err := io.Copy(chunkPart, r); err != nil {
		return nil, fmt.Errorf("upload: copy chunk data: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("upload: close multipart: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/files/upload", &buf)
	if err != nil {
		return nil, fmt.Errorf("upload: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, readAPIError(resp)
	}

	var entry FileEntry
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, fmt.Errorf("upload: decode: %w", err)
	}
	return &entry, nil
}

// DownloadFile downloads the file with the given ID and returns a reader for
// the body. The caller must close the returned ReadCloser.
func (c *Client) DownloadFile(fileID string) (io.ReadCloser, error) {
	u := c.baseURL + "/api/v1/files/" + url.PathEscape(fileID) + "/download"
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, readAPIError(resp)
	}
	return resp.Body, nil
}

// DeleteFile soft-deletes (trashes) the file with the given ID.
func (c *Client) DeleteFile(fileID string) error {
	u := c.baseURL + "/api/v1/files/" + url.PathEscape(fileID)
	req, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return readAPIError(resp)
	}
	return nil
}

// CreateFolder creates a new folder with the given name under parentID.
// Pass empty string for parentID to create in the root.
func (c *Client) CreateFolder(name string, parentID string) (*FileEntry, error) {
	body := map[string]string{
		"name_encrypted": name,
	}
	if parentID != "" {
		body["parent_id"] = parentID
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("create folder: marshal: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/files/folder", bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("create folder: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("create folder: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, readAPIError(resp)
	}

	var entry FileEntry
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, fmt.Errorf("create folder: decode: %w", err)
	}
	return &entry, nil
}

// GetFile returns the metadata for a single file by ID.
func (c *Client) GetFile(fileID string) (*FileEntry, error) {
	u := c.baseURL + "/api/v1/files/" + url.PathEscape(fileID)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("get file: %w", err)
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("get file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, readAPIError(resp)
	}

	var entry FileEntry
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, fmt.Errorf("get file: decode: %w", err)
	}
	return &entry, nil
}

// readAPIError reads the standard {"error": "..."} response body and returns
// it as a Go error that includes the HTTP status code.
func readAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var apiErr struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &apiErr) == nil && apiErr.Error != "" {
		return fmt.Errorf("api %d: %s", resp.StatusCode, apiErr.Error)
	}
	return fmt.Errorf("api %d: %s", resp.StatusCode, string(body))
}

// joinPath is a helper that joins path segments, handling leading/trailing slashes.
func joinPath(base string, segments ...string) string {
	return path.Join(append([]string{base}, segments...)...)
}
