package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var defaultSkipDirs = map[string]struct{}{
	".git":        {},
	".hg":         {},
	".svn":        {},
	"node_modules": {},
	"vendor":      {},
	"dist":        {},
	"build":       {},
	"target":      {},
}

type stats struct {
	Files int64
	Lines int64
	Bytes int64
}

func main() {
	skipDirs := flag.String("skip", "", "comma-separated directory names to skip (in addition to defaults)")
	countExts := flag.String("count", "", "comma-separated file extensions to count (example: go,md)")
	includeHidden := flag.Bool("include-hidden", true, "include hidden files/directories (except skipped directories)")
	flag.Parse()

	skip := make(map[string]struct{}, len(defaultSkipDirs))
	for k := range defaultSkipDirs {
		skip[k] = struct{}{}
	}
	if *skipDirs != "" {
		for _, d := range strings.Split(*skipDirs, ",") {
			d = strings.TrimSpace(d)
			if d == "" {
				continue
			}
			skip[d] = struct{}{}
		}
	}

	count := parseExtensions(*countExts)

	result, err := scan(".", skip, count, *includeHidden)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Files: %d\n", result.Files)
	fmt.Printf("Lines: %d\n", result.Lines)
	fmt.Printf("Bytes: %d\n", result.Bytes)
}

func scan(root string, skip map[string]struct{}, count map[string]struct{}, includeHidden bool) (stats, error) {
	var s stats

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		name := d.Name()
		if d.IsDir() {
			if _, ok := skip[name]; ok && path != root {
				return filepath.SkipDir
			}
			if !includeHidden && strings.HasPrefix(name, ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}

		if !d.Type().IsRegular() {
			return nil
		}
		if !includeHidden && strings.HasPrefix(name, ".") {
			return nil
		}
		if len(count) > 0 {
			ext := strings.ToLower(filepath.Ext(name))
			if _, ok := count[ext]; !ok {
				return nil
			}
		}

		lines, size, err := countLines(path)
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return nil
			}
			return err
		}

		s.Files++
		s.Lines += lines
		s.Bytes += size
		return nil
	})

	return s, err
}

func parseExtensions(raw string) map[string]struct{} {
	if strings.TrimSpace(raw) == "" {
		return map[string]struct{}{}
	}

	exts := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		ext := strings.TrimSpace(strings.ToLower(part))
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		exts[ext] = struct{}{}
	}
	return exts
}

func countLines(path string) (int64, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	var (
		lines      int64
		size       int64
		lastByteNL = true
		buf        = make([]byte, 32*1024)
	)

	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			size += int64(n)
			for _, b := range chunk {
				if b == '\n' {
					lines++
					lastByteNL = true
				} else {
					lastByteNL = false
				}
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return 0, 0, readErr
		}
	}

	if size > 0 && !lastByteNL {
		lines++
	}

	return lines, size, nil
}
