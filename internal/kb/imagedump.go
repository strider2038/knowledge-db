package kb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/muonsoft/errors"
	"github.com/spf13/afero"
)

// DryRunResult holds URL and target path for dry-run output.
type DryRunResult struct {
	URL        string
	TargetPath string
}

// DownloadError describes a failed image download.
type DownloadError struct {
	URL string
	Err error
}

const maxImageSize = 10 * 1024 * 1024 // 10 MB

// ImageRef describes a markdown image or link reference to a remote image.
type ImageRef struct {
	URL           string
	Alt           string
	IsImageSyntax bool // ![alt](url) vs [text](url)
}

// imageExtPattern matches URLs ending with image extensions (case-insensitive).
var imageExtPattern = regexp.MustCompile(`(?i)\.(png|jpg|jpeg|gif|webp|svg)(?:\?[^)]*)?$`)

// ExtractImageURLs finds all remote image URLs in markdown body.
// Supports ![alt](url) and [text](url) with image extensions.
// Ignores local paths (no http/https scheme).
func ExtractImageURLs(body string) []ImageRef {
	var refs []ImageRef
	seen := make(map[string]struct{})

	// ![alt](url)
	reImg := regexp.MustCompile(`!\[([^\]]*)\]\((https?://[^)]+)\)`)
	for _, m := range reImg.FindAllStringSubmatch(body, -1) {
		if len(m) >= 3 && imageExtPattern.MatchString(m[2]) {
			if _, ok := seen[m[2]]; !ok {
				seen[m[2]] = struct{}{}
				refs = append(refs, ImageRef{URL: m[2], Alt: m[1], IsImageSyntax: true})
			}
		}
	}

	// [text](url) with image extension
	reLink := regexp.MustCompile(`\[([^\]]*)\]\((https?://[^)]+)\)`)
	for _, m := range reLink.FindAllStringSubmatch(body, -1) {
		if len(m) >= 3 && imageExtPattern.MatchString(m[2]) {
			if _, ok := seen[m[2]]; !ok {
				seen[m[2]] = struct{}{}
				refs = append(refs, ImageRef{URL: m[2], Alt: m[1], IsImageSyntax: false})
			}
		}
	}

	return refs
}

// ImageFilename returns a filename for the image: first 12 chars of SHA256(url) + extension.
func ImageFilename(imageURL string) (string, error) {
	u, err := url.Parse(imageURL)
	if err != nil {
		return "", errors.Errorf("parse url: %w", err)
	}
	path := u.Path
	if path == "" {
		path = u.Opaque
	}
	ext := ""
	for _, e := range []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg"} {
		if strings.HasSuffix(strings.ToLower(path), e) {
			ext = e

			break
		}
	}
	if ext == "" {
		return "", errors.Errorf("unsupported image extension in url: %s", imageURL)
	}
	hash := sha256.Sum256([]byte(imageURL))

	return hex.EncodeToString(hash[:])[:12] + ext, nil
}

// DownloadImage fetches an image from URL with Content-Type check and size limit.
func DownloadImage(ctx context.Context, client *http.Client, imageURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, errors.Errorf("create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Errorf("fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("fetch image: status %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "" && !strings.HasPrefix(ct, "image/") {
		return nil, errors.Errorf("fetch image: content-type %q is not image", ct)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageSize+1))
	if err != nil {
		return nil, errors.Errorf("read image: %w", err)
	}
	if int64(len(data)) > maxImageSize {
		return nil, errors.Errorf("fetch image: size exceeds %d bytes", maxImageSize)
	}

	return data, nil
}

// OnDownloadFunc is called after each successful image download (url, targetPath, size in bytes).
type OnDownloadFunc func(url, targetPath string, size int64)

// RunDumpImages downloads remote images from markdown, saves to {themePath}/{slug}/images/,
// replaces URLs in body, and updates frontmatter.updated.
// When dryRun is true, returns dryRunResults for the caller to print; no files are modified.
// Failed downloads are returned in downloadErrors for the caller to log.
// If onDownload is not nil, it is called after each successful download.
func RunDumpImages(ctx context.Context, fs afero.Fs, client *http.Client, basePath, themePath, slug string, dryRun bool, onDownload OnDownloadFunc) (bool, []DownloadError, []DryRunResult, error) {
	stemPath := filepath.Join(basePath, filepath.FromSlash(themePath), slug)
	mdPath := stemPath + ".md"

	data, err := afero.ReadFile(fs, mdPath)
	if err != nil {
		return false, nil, nil, errors.Errorf("read node file: %w", err)
	}

	var matter map[string]any
	rest, err := frontmatter.Parse(strings.NewReader(string(data)), &matter)
	if err != nil {
		return false, nil, nil, errors.Errorf("parse frontmatter: %w", err)
	}
	body := string(rest)

	refs := ExtractImageURLs(body)
	if len(refs) == 0 {
		return false, nil, nil, nil
	}

	if dryRun {
		var results []DryRunResult
		var errs []DownloadError
		for _, r := range refs {
			fn, e := ImageFilename(r.URL)
			if e != nil {
				errs = append(errs, DownloadError{URL: r.URL, Err: e})

				continue
			}
			relPath := filepath.ToSlash(filepath.Join(slug, "images", fn))
			results = append(results, DryRunResult{URL: r.URL, TargetPath: relPath})
		}

		return false, errs, results, nil
	}

	imagesDir := filepath.Join(stemPath, "images")
	if err := fs.MkdirAll(imagesDir, 0o755); err != nil {
		return false, nil, nil, errors.Errorf("create images dir: %w", err)
	}

	var downloadErrs []DownloadError
	urlToFilename := make(map[string]string)
	for _, r := range refs {
		fn, err := ImageFilename(r.URL)
		if err != nil {
			downloadErrs = append(downloadErrs, DownloadError{URL: r.URL, Err: err})

			continue
		}

		destPath := filepath.Join(imagesDir, fn)
		if _, statErr := fs.Stat(destPath); statErr == nil {
			urlToFilename[r.URL] = fn

			continue
		}

		imgData, err := DownloadImage(ctx, client, r.URL)
		if err != nil {
			downloadErrs = append(downloadErrs, DownloadError{URL: r.URL, Err: err})

			continue
		}

		if err := afero.WriteFile(fs, destPath, imgData, 0o644); err != nil {
			downloadErrs = append(downloadErrs, DownloadError{URL: r.URL, Err: err})

			continue
		}
		urlToFilename[r.URL] = fn
		if onDownload != nil {
			relPath := filepath.ToSlash(filepath.Join(slug, "images", fn))
			onDownload(r.URL, relPath, int64(len(imgData)))
		}
	}

	if len(urlToFilename) == 0 {
		return false, downloadErrs, nil, nil
	}

	newBody := body
	for urlStr, fn := range urlToFilename {
		relPath := filepath.ToSlash(filepath.Join(slug, "images", fn))
		newBody = strings.ReplaceAll(newBody, "("+urlStr+")", "("+relPath+")")
	}

	modified := false
	if newBody != body {
		matter["updated"] = time.Now().UTC().Format(time.RFC3339)
		fmBytes, err := FormatFrontmatter(matter)
		if err != nil {
			return false, downloadErrs, nil, errors.Errorf("format frontmatter: %w", err)
		}
		out := string(fmBytes) + newBody
		if err := afero.WriteFile(fs, mdPath, []byte(out), 0o644); err != nil {
			return false, downloadErrs, nil, errors.Errorf("write node file: %w", err)
		}
		modified = true
	}

	return modified, downloadErrs, nil, nil
}
