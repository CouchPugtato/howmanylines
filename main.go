package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var defaultSkipDirs = map[string]struct{}{
	".git":         {},
	".hg":          {},
	".svn":         {},
	"node_modules": {},
	"vendor":       {},
	"dist":         {},
	"build":        {},
	"target":       {},
}

var defaultSkipFiles = map[string]struct{}{
	"package-lock.json": {},
	"yarn.lock":         {},
	"pnpm-lock.yaml":    {},
	"bun.lockb":         {},
	"cargo.lock":        {},
	"cargo.toml":        {},
	"go.sum":            {},
	"poetry.lock":       {},
	"pipfile.lock":      {},
	"composer.lock":     {},
	"gemfile.lock":      {},
}

type stats struct {
	Files   int64
	Lines   int64
	Bytes   int64
	PerFile []fileStat
}

type fileStat struct {
	Path  string
	Lines int64
	Bytes int64
}

type extStat struct {
	Ext   string
	Files int64
	Lines int64
}

func main() {
	skipDirs := flag.String("skip", "", "comma-separated directory names to skip (in addition to defaults)")
	countExts := flag.String("count", "", "comma-separated file extensions to count (example: go,md)")
	includeHidden := flag.Bool("include-hidden", false, "include hidden files/directories (except skipped directories)")
	rank := flag.Bool("rank", false, "show leaderboards for files and extensions by line count")
	flag.Parse()
	resolvedTop, err := resolveTop(*rank, flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

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

	fmt.Printf("Files: %s\n", formatWithCommas(result.Files))
	fmt.Printf("Lines: %s\n", formatWithCommas(result.Lines))
	fmt.Printf("Bytes: %s\n", formatWithCommas(result.Bytes))
	if *rank {
		printFileLeaderboard(result.PerFile, resolvedTop)
		printExtensionLeaderboard(result.PerFile, resolvedTop)
	}
}

func resolveTop(rank bool, args []string) (int, error) {
	top := 3
	if len(args) == 0 {
		return top, nil
	}
	if !rank {
		return 0, fmt.Errorf("unexpected argument: %s", args[0])
	}
	n, err := strconv.Atoi(args[0])
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid rank value: %s (must be a positive integer)", args[0])
	}
	if len(args) > 1 {
		return 0, fmt.Errorf("unexpected argument: %s", args[1])
	}
	return n, nil
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
		lowerName := strings.ToLower(name)
		ext := strings.ToLower(filepath.Ext(name))
		if len(count) > 0 {
			if _, ok := count[ext]; !ok {
				return nil
			}
		} else {
			if _, ok := defaultSkipFiles[lowerName]; ok {
				return nil
			}
			if ext == "" || ext == ".exe" {
				return nil
			}
			if !isLikelyTextFile(path) {
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
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			relPath = path
		}
		s.PerFile = append(s.PerFile, fileStat{
			Path:  relPath,
			Lines: lines,
			Bytes: size,
		})
		return nil
	})

	return s, err
}

func printFileLeaderboard(entries []fileStat, top int) {
	sorted := append([]fileStat(nil), entries...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Lines != sorted[j].Lines {
			return sorted[i].Lines > sorted[j].Lines
		}
		return sorted[i].Path < sorted[j].Path
	})
	requested := top
	if requested > len(sorted) {
		top = len(sorted)
	} else {
		top = requested
	}

	fmt.Println()
	if top < requested {
		fmt.Printf("Top %d Files (showing %d):\n", requested, top)
	} else {
		fmt.Printf("Top %d Files:\n", requested)
	}
	rankWidth := len(strconv.Itoa(maxInt(1, top)))
	lineWidth := maxLineWidthFiles(sorted[:top])
	for i, e := range sorted[:top] {
		fmt.Printf("%*d. %*s lines  %s\n", rankWidth, i+1, lineWidth, formatWithCommas(e.Lines), e.Path)
	}
}

func isLikelyTextFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	const sniffSize = 8192
	buf := make([]byte, sniffSize)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}
	data := buf[:n]
	if len(data) == 0 {
		return true
	}

	if bytes.IndexByte(data, 0) != -1 {
		return false
	}

	nonText := 0
	for _, b := range data {
		if b == '\n' || b == '\r' || b == '\t' {
			continue
		}
		if b >= 32 && b <= 126 {
			continue
		}
		if b >= 128 {
			continue
		}
		nonText++
	}

	return float64(nonText)/float64(len(data)) <= 0.30
}

func printExtensionLeaderboard(entries []fileStat, top int) {
	byExt := make(map[string]extStat)
	for _, e := range entries {
		ext := strings.ToLower(filepath.Ext(e.Path))
		if ext == "" {
			ext = "(no extension)"
		}
		curr := byExt[ext]
		curr.Ext = ext
		curr.Files++
		curr.Lines += e.Lines
		byExt[ext] = curr
	}

	sorted := make([]extStat, 0, len(byExt))
	for _, e := range byExt {
		sorted = append(sorted, e)
	}

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Lines != sorted[j].Lines {
			return sorted[i].Lines > sorted[j].Lines
		}
		if sorted[i].Files != sorted[j].Files {
			return sorted[i].Files > sorted[j].Files
		}
		return sorted[i].Ext < sorted[j].Ext
	})
	requested := top
	if requested > len(sorted) {
		top = len(sorted)
	} else {
		top = requested
	}

	fmt.Println()
	if top < requested {
		fmt.Printf("Top %d Extensions (showing %d):\n", requested, top)
	} else {
		fmt.Printf("Top %d Extensions:\n", requested)
	}
	rankWidth := len(strconv.Itoa(maxInt(1, top)))
	lineWidth := maxLineWidthExts(sorted[:top])
	for i, e := range sorted[:top] {
		fmt.Printf("%*d. %*s lines  %s\n", rankWidth, i+1, lineWidth, formatWithCommas(e.Lines), e.Ext)
	}
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

func formatWithCommas(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}

	sign := ""
	if s[0] == '-' {
		sign = "-"
		s = s[1:]
	}

	rem := len(s) % 3
	if rem == 0 {
		rem = 3
	}

	var b strings.Builder
	b.Grow(len(s) + len(s)/3)
	b.WriteString(sign)
	b.WriteString(s[:rem])
	for i := rem; i < len(s); i += 3 {
		b.WriteByte(',')
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxLineWidthFiles(entries []fileStat) int {
	width := 1
	for _, e := range entries {
		n := len(formatWithCommas(e.Lines))
		if n > width {
			width = n
		}
	}
	return width
}

func maxLineWidthExts(entries []extStat) int {
	width := 1
	for _, e := range entries {
		n := len(formatWithCommas(e.Lines))
		if n > width {
			width = n
		}
	}
	return width
}
