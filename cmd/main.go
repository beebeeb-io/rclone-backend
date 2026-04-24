// Beebeeb rclone backend — standalone test CLI
// Copyright (C) 2026 Beebeeb
// SPDX-License-Identifier: AGPL-3.0-or-later

// Command bb-rclone-test exercises the Beebeeb rclone backend without
// requiring a full rclone build.
//
// Usage:
//
//	BB_TOKEN=<token> go run ./cmd --list /
//	BB_TOKEN=<token> go run ./cmd --upload ./test.txt /test.txt
//	BB_TOKEN=<token> go run ./cmd --download /test.txt ./downloaded.txt
//	BB_TOKEN=<token> go run ./cmd --mkdir /backup/2026-04
//	BB_TOKEN=<token> go run ./cmd --delete /test.txt
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	beebeeb "github.com/beebeeb-io/rclone-backend"
)

func main() {
	listDir := flag.String("list", "", "List files in the given remote directory")
	uploadSrc := flag.String("upload", "", "Upload a local file (use with positional arg for remote path)")
	downloadSrc := flag.String("download", "", "Download a remote file (use with positional arg for local path)")
	mkdirPath := flag.String("mkdir", "", "Create a remote directory")
	deletePath := flag.String("delete", "", "Delete a remote file")
	flag.Parse()

	cfg, err := beebeeb.ConfigFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fs := beebeeb.NewFsFromConfig("beebeeb", "", cfg)

	switch {
	case *listDir != "":
		cmdList(fs, *listDir)

	case *uploadSrc != "":
		if flag.NArg() < 1 {
			fmt.Fprintf(os.Stderr, "usage: --upload <local-path> <remote-path>\n")
			os.Exit(1)
		}
		cmdUpload(fs, *uploadSrc, flag.Arg(0))

	case *downloadSrc != "":
		if flag.NArg() < 1 {
			fmt.Fprintf(os.Stderr, "usage: --download <remote-path> <local-path>\n")
			os.Exit(1)
		}
		cmdDownload(fs, *downloadSrc, flag.Arg(0))

	case *mkdirPath != "":
		cmdMkdir(fs, *mkdirPath)

	case *deletePath != "":
		cmdDelete(fs, *deletePath)

	default:
		flag.Usage()
		os.Exit(1)
	}
}

func cmdList(fs *beebeeb.Fs, dir string) {
	objects, err := fs.List(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(objects) == 0 {
		fmt.Println("(empty)")
		return
	}

	for _, obj := range objects {
		kind := "file"
		if obj.IsDir() {
			kind = "dir "
		}
		fmt.Printf("  %s  %10d  %s  %s\n", kind, obj.Size(), obj.ModTime().Format(time.RFC3339), obj.Remote())
	}
	fmt.Printf("\n  %d items\n", len(objects))
}

func cmdUpload(fs *beebeeb.Fs, localPath, remotePath string) {
	f, err := os.Open(localPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	src := &beebeeb.SimpleObjectInfo{
		Path:  remotePath,
		Bytes: info.Size(),
		ModAt: info.ModTime(),
	}

	obj, err := fs.Put(f, src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("uploaded %s -> %s (%d bytes, id=%s)\n", localPath, remotePath, obj.Size(), obj.ID())
}

func cmdDownload(fs *beebeeb.Fs, remotePath, localPath string) {
	rc, err := fs.Get(remotePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer rc.Close()

	out, err := os.Create(localPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	n, err := io.Copy(out, rc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("downloaded %s -> %s (%d bytes)\n", remotePath, localPath, n)
}

func cmdMkdir(fs *beebeeb.Fs, dir string) {
	if err := fs.Mkdir(dir); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("created directory: %s\n", dir)
}

func cmdDelete(fs *beebeeb.Fs, remotePath string) {
	if err := fs.Remove(remotePath); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("deleted: %s\n", remotePath)
}
