// Beebeeb rclone backend — core Fs and Object implementation
// Copyright (C) 2026 Beebeeb
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package beebeeb implements an rclone backend for the Beebeeb encrypted vault.
//
// Because rclone backends are compiled into the rclone binary (or loaded as
// plugins), this package mirrors the rclone fs.Fs and fs.Object interfaces as
// standalone types. To register with rclone, add this package as an import in
// an rclone fork or plugin build:
//
//	import _ "github.com/beebeeb-io/rclone-backend"
//
// The backend is registered under the name "bb" so that users can create
// remotes with:
//
//	rclone config create beebeeb bb token=<session_token> api_url=https://api.beebeeb.io
//
// Then use standard rclone commands:
//
//	rclone ls beebeeb:
//	rclone copy ./local-dir beebeeb:backup/
//	rclone sync beebeeb:documents/ ./local-docs/
//	rclone mount beebeeb: ~/vault --vfs-cache-mode full
package beebeeb

import (
	"fmt"
	"io"
	"path"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Interfaces — these mirror rclone's fs.Fs, fs.Object, and fs.ObjectInfo.
// When compiled into rclone, these would be replaced by the real interfaces.
// ---------------------------------------------------------------------------

// ObjectInfo describes an object to be uploaded.
type ObjectInfo interface {
	Remote() string
	Size() int64
	ModTime() time.Time
}

// Object represents a remote file.
type Object interface {
	ObjectInfo
	Open() (io.ReadCloser, error)
	Remove() error
	ID() string
	IsDir() bool
	MimeType() string
}

// Fs is the rclone filesystem interface for a Beebeeb vault.
type Fs struct {
	name   string // remote name (e.g. "beebeeb")
	root   string // root path within the vault
	client *Client

	// dirCache maps directory paths to their API IDs so that we can resolve
	// nested paths like "archive/pg/2026-04" without repeated lookups.
	dirCache map[string]string
}

// ---------------------------------------------------------------------------
// Construction
// ---------------------------------------------------------------------------

// NewFs creates a new Beebeeb backend instance.
//
// name is the rclone remote name (e.g. "beebeeb").
// root is the path within the vault to treat as the root (can be empty for /).
// config is a map of configuration options from the rclone config file.
func NewFs(name, root string, config map[string]string) (*Fs, error) {
	cfg, err := ConfigFromMap(config)
	if err != nil {
		return nil, fmt.Errorf("beebeeb: %w", err)
	}

	return &Fs{
		name:     name,
		root:     strings.Trim(root, "/"),
		client:   NewClient(cfg),
		dirCache: make(map[string]string),
	}, nil
}

// NewFsFromConfig creates a new Beebeeb backend instance from an explicit Config.
func NewFsFromConfig(name, root string, cfg *Config) *Fs {
	return &Fs{
		name:     name,
		root:     strings.Trim(root, "/"),
		client:   NewClient(cfg),
		dirCache: make(map[string]string),
	}
}

// Name returns the remote name.
func (f *Fs) Name() string { return f.name }

// Root returns the root path.
func (f *Fs) Root() string { return f.root }

// ---------------------------------------------------------------------------
// Directory operations
// ---------------------------------------------------------------------------

// resolveDir walks down the path components from the root and returns the
// API ID of the deepest folder. It caches results in dirCache.
func (f *Fs) resolveDir(dir string) (string, error) {
	dir = strings.Trim(dir, "/")
	if dir == "" {
		return "", nil // root folder has no parent_id
	}

	// Check cache.
	if id, ok := f.dirCache[dir]; ok {
		return id, nil
	}

	// Walk path components.
	parts := strings.Split(dir, "/")
	parentID := ""
	walked := ""
	for _, part := range parts {
		if walked != "" {
			walked += "/"
		}
		walked += part

		if id, ok := f.dirCache[walked]; ok {
			parentID = id
			continue
		}

		// List parent and find the folder by name.
		entries, err := f.client.ListFiles(parentID)
		if err != nil {
			return "", fmt.Errorf("resolve %q: %w", walked, err)
		}

		found := false
		for _, e := range entries {
			if e.IsFolder && e.NameEncrypted == part {
				f.dirCache[walked] = e.ID
				parentID = e.ID
				found = true
				break
			}
		}
		if !found {
			return "", fmt.Errorf("directory not found: %s", walked)
		}
	}
	return parentID, nil
}

// List returns all files and folders in the given directory relative to root.
func (f *Fs) List(dir string) ([]Object, error) {
	fullDir := path.Join(f.root, dir)
	parentID, err := f.resolveDir(fullDir)
	if err != nil {
		return nil, err
	}

	entries, err := f.client.ListFiles(parentID)
	if err != nil {
		return nil, err
	}

	objects := make([]Object, 0, len(entries))
	for _, e := range entries {
		obj := &remoteObject{
			fs:    f,
			entry: e,
			path:  path.Join(dir, e.NameEncrypted),
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

// Mkdir creates the directory (and any parents) at the given path.
func (f *Fs) Mkdir(dir string) error {
	fullDir := path.Join(f.root, dir)
	parts := strings.Split(strings.Trim(fullDir, "/"), "/")
	if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
		return nil // root always exists
	}

	parentID := ""
	walked := ""
	for _, part := range parts {
		if walked != "" {
			walked += "/"
		}
		walked += part

		if id, ok := f.dirCache[walked]; ok {
			parentID = id
			continue
		}

		// Check if the folder already exists.
		entries, err := f.client.ListFiles(parentID)
		if err != nil {
			return fmt.Errorf("mkdir %q: %w", walked, err)
		}

		found := false
		for _, e := range entries {
			if e.IsFolder && e.NameEncrypted == part {
				f.dirCache[walked] = e.ID
				parentID = e.ID
				found = true
				break
			}
		}
		if found {
			continue
		}

		// Create the folder.
		entry, err := f.client.CreateFolder(part, parentID)
		if err != nil {
			return fmt.Errorf("mkdir %q: %w", walked, err)
		}
		f.dirCache[walked] = entry.ID
		parentID = entry.ID
	}
	return nil
}

// ---------------------------------------------------------------------------
// File operations
// ---------------------------------------------------------------------------

// Put uploads a file from in, using src for metadata (remote path, size).
func (f *Fs) Put(in io.Reader, src ObjectInfo) (Object, error) {
	remote := src.Remote()
	dir := path.Dir(remote)
	name := path.Base(remote)

	// Ensure the parent directory exists.
	fullDir := path.Join(f.root, dir)
	if fullDir != "" && fullDir != "." {
		if err := f.Mkdir(dir); err != nil {
			return nil, fmt.Errorf("put: mkdir parent: %w", err)
		}
	}

	parentID, err := f.resolveDir(fullDir)
	if err != nil {
		return nil, fmt.Errorf("put: resolve parent: %w", err)
	}

	entry, err := f.client.UploadFile(name, parentID, src.Size(), "", in)
	if err != nil {
		return nil, err
	}

	return &remoteObject{
		fs:    f,
		entry: *entry,
		path:  remote,
	}, nil
}

// Get opens a file for reading by its path relative to root.
// The caller must close the returned ReadCloser.
func (f *Fs) Get(remotePath string) (io.ReadCloser, error) {
	obj, err := f.findObject(remotePath)
	if err != nil {
		return nil, err
	}
	return f.client.DownloadFile(obj.entry.ID)
}

// Remove deletes a file by its path relative to root.
func (f *Fs) Remove(remotePath string) error {
	obj, err := f.findObject(remotePath)
	if err != nil {
		return err
	}
	return f.client.DeleteFile(obj.entry.ID)
}

// findObject locates a file by its path and returns its remoteObject.
func (f *Fs) findObject(remotePath string) (*remoteObject, error) {
	dir := path.Dir(remotePath)
	name := path.Base(remotePath)

	fullDir := path.Join(f.root, dir)
	parentID, err := f.resolveDir(fullDir)
	if err != nil {
		return nil, fmt.Errorf("find %q: %w", remotePath, err)
	}

	entries, err := f.client.ListFiles(parentID)
	if err != nil {
		return nil, fmt.Errorf("find %q: %w", remotePath, err)
	}

	for _, e := range entries {
		if e.NameEncrypted == name {
			return &remoteObject{
				fs:    f,
				entry: e,
				path:  remotePath,
			}, nil
		}
	}
	return nil, fmt.Errorf("file not found: %s", remotePath)
}

// ---------------------------------------------------------------------------
// remoteObject — implements Object
// ---------------------------------------------------------------------------

type remoteObject struct {
	fs    *Fs
	entry FileEntry
	path  string
}

func (o *remoteObject) Remote() string      { return o.path }
func (o *remoteObject) Size() int64         { return o.entry.SizeBytes }
func (o *remoteObject) ModTime() time.Time  { return o.entry.UpdatedAt }
func (o *remoteObject) ID() string          { return o.entry.ID }
func (o *remoteObject) IsDir() bool         { return o.entry.IsFolder }
func (o *remoteObject) MimeType() string    { return o.entry.MimeType }

func (o *remoteObject) Open() (io.ReadCloser, error) {
	return o.fs.client.DownloadFile(o.entry.ID)
}

func (o *remoteObject) Remove() error {
	return o.fs.client.DeleteFile(o.entry.ID)
}

// ---------------------------------------------------------------------------
// simpleObjectInfo — a minimal ObjectInfo for Put
// ---------------------------------------------------------------------------

// SimpleObjectInfo is a minimal ObjectInfo implementation for use with Put.
type SimpleObjectInfo struct {
	Path    string
	Bytes   int64
	ModAt   time.Time
}

func (s *SimpleObjectInfo) Remote() string     { return s.Path }
func (s *SimpleObjectInfo) Size() int64        { return s.Bytes }
func (s *SimpleObjectInfo) ModTime() time.Time { return s.ModAt }
