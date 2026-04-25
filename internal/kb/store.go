package kb

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/muonsoft/errors"
	"github.com/spf13/afero"
)

// translationFilePattern — *.[a-z]{2}.md (переводы не включаются в список).
var translationFilePattern = regexp.MustCompile(`\.[a-z]{2}\.md$`)

// Store — хранилище базы знаний с абстракцией файловой системы.
// Позволяет использовать in-memory fs в тестах (afero.MemMapFs).
type Store struct {
	fs afero.Fs
}

// NewStore создаёт Store с указанной файловой системой.
// Для production: afero.NewOsFs()
// Для тестов: afero.NewMemMapFs().
func NewStore(fs afero.Fs) *Store {
	return &Store{fs: fs}
}

// Validate проверяет структуру базы: темы 2–3 уровня, узлы как .md файлы с frontmatter.
//
//nolint:gocognit // single walk: depth, hidden dirs, attachment dirs, node checks
func (s *Store) Validate(ctx context.Context, basePath string) ([]ValidationError, error) {
	basePath = filepath.Clean(basePath)
	info, err := s.fs.Stat(basePath)
	if err != nil {
		return nil, errors.Errorf("validate base: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.Errorf("validate base: %w", ErrInvalidPath)
	}

	allNodes, err := s.ListAllNodes(ctx, basePath)
	if err != nil {
		return nil, errors.Errorf("validate: list nodes: %w", err)
	}
	nodePaths := make(map[string]struct{})
	for _, n := range allNodes {
		nodePaths[n.Path] = struct{}{}
	}

	var violations []ValidationError
	err = afero.Walk(s.fs, basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(basePath, path)
		if rel == "." {
			return nil
		}
		relSlash := filepath.ToSlash(rel)
		if pathHasHiddenSegment(relSlash) {
			if info.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}
		parts := strings.Split(relSlash, "/")
		depth := len(parts)

		if info.IsDir() {
			// Если рядом существует {dirname}.md — это директория вложений, пропускаем.
			if _, statErr := s.fs.Stat(path + ".md"); statErr == nil {
				return filepath.SkipDir
			}
			if depth > maxDepth {
				violations = append(violations, ValidationError{Path: rel, Message: "theme depth exceeds 2-3 levels"})

				return filepath.SkipDir
			}

			return nil
		}

		// Файл: проверяем только .md узлы.
		if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}
		stemRel := strings.TrimSuffix(filepath.ToSlash(rel), ".md")
		if strings.HasSuffix(info.Name(), ".ru.md") {
			if err := s.validateTranslationFile(basePath, path, stemRel, nodePaths, &violations); err != nil {
				return err
			}
		} else {
			if err := s.validateNode(path, stemRel, &violations); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, errors.Errorf("validate base: %w", err)
	}

	return violations, nil
}

// ReadTree возвращает дерево тем и подтем базы знаний.
func (s *Store) ReadTree(ctx context.Context, basePath string) (*TreeNode, error) {
	basePath = filepath.Clean(basePath)
	info, err := s.fs.Stat(basePath)
	if err != nil {
		return nil, errors.Errorf("read tree: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.Errorf("read tree: %w", ErrInvalidPath)
	}

	root := &TreeNode{Name: "", Path: ""}
	if err := s.buildTree(basePath, basePath, "", root, 0); err != nil {
		return nil, err
	}

	return root, nil
}

// ListNodes возвращает список узлов по пути темы.
func (s *Store) ListNodes(ctx context.Context, basePath, themePath string) ([]*TreeNode, error) {
	basePath = filepath.Clean(basePath)
	themeForCheck := filepath.ToSlash(filepath.Clean(filepath.FromSlash(themePath)))
	if pathHasHiddenSegment(themeForCheck) {
		return nil, errors.Errorf("list nodes: %w", ErrNodeNotFound)
	}
	fullPath := filepath.Join(basePath, filepath.FromSlash(themePath))
	info, err := s.fs.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Errorf("list nodes: %w", ErrNodeNotFound)
		}

		return nil, errors.Errorf("list nodes: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.Errorf("list nodes: %w", ErrInvalidPath)
	}

	var nodes []*TreeNode
	entries, err := s.readDir(fullPath)
	if err != nil {
		return nil, errors.Errorf("list nodes: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		if strings.HasPrefix(name, ".") {
			continue
		}
		slug := strings.TrimSuffix(name, ".md")
		nodeRel := filepath.ToSlash(filepath.Join(themePath, slug))
		nodes = append(nodes, &TreeNode{
			Name: slug,
			Path: nodeRel,
		})
	}

	return nodes, nil
}

// IsNode проверяет, является ли стем-путь узлом (существует {stem}.md).
func (s *Store) IsNode(stemPath string) bool {
	return s.isNode(stemPath)
}

// ListAllNodes рекурсивно возвращает все узлы базы знаний.
func (s *Store) ListAllNodes(ctx context.Context, basePath string) ([]*TreeNode, error) {
	basePath = filepath.Clean(basePath)
	var nodes []*TreeNode
	err := afero.Walk(s.fs, basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(basePath, path)
		relSlash := filepath.ToSlash(rel)
		if info.IsDir() {
			if rel != "." && pathHasHiddenSegment(relSlash) {
				return filepath.SkipDir
			}

			return nil
		}
		name := info.Name()
		if !strings.HasSuffix(name, ".md") || strings.HasPrefix(name, ".") {
			return nil
		}
		stemRel := strings.TrimSuffix(relSlash, ".md")
		if pathHasHiddenSegment(stemRel) {
			return nil
		}
		slug := strings.TrimSuffix(name, ".md")
		nodes = append(nodes, &TreeNode{
			Name: slug,
			Path: stemRel,
		})

		return nil
	})
	if err != nil {
		return nil, errors.Errorf("list all nodes: %w", err)
	}

	return nodes, nil
}

// ListNodesWithOptions возвращает список узлов с метаданными, фильтрацией, поиском, сортировкой и пагинацией.
// Переводы (*.[a-z]{2}.md) не включаются в список.
//
//nolint:gocognit,gocyclo,maintidx // walk + filters + sort in one pass
func (s *Store) ListNodesWithOptions(ctx context.Context, basePath string, opts ListNodesOptions) ([]*NodeListItem, int, error) {
	basePath = filepath.Clean(basePath)
	info, err := s.fs.Stat(basePath)
	if err != nil {
		return nil, 0, errors.Errorf("list nodes: %w", err)
	}
	if !info.IsDir() {
		return nil, 0, errors.Errorf("list nodes: %w", ErrInvalidPath)
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := max(opts.Offset, 0)
	sortField := opts.Sort
	if sortField == "" {
		sortField = "title"
	}
	order := opts.Order
	if order == "" {
		order = "asc"
	}

	var items []*NodeListItem
	err = afero.Walk(s.fs, basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(basePath, path)
		relSlash := filepath.ToSlash(rel)
		if info.IsDir() {
			if rel != "." && pathHasHiddenSegment(relSlash) {
				return filepath.SkipDir
			}

			return nil
		}
		name := info.Name()
		if !strings.HasSuffix(name, ".md") || strings.HasPrefix(name, ".") {
			return nil
		}
		if translationFilePattern.MatchString(name) {
			return nil
		}
		stemRel := strings.TrimSuffix(relSlash, ".md")
		if stemRel == "" {
			return nil
		}
		if pathHasHiddenSegment(stemRel) {
			return nil
		}

		if opts.Path != "" {
			if stemRel != opts.Path && !strings.HasPrefix(stemRel, opts.Path+"/") {
				return nil
			}
			if !opts.Recursive && stemRel != opts.Path {
				rest := strings.TrimPrefix(stemRel, opts.Path+"/")
				if strings.Contains(rest, "/") {
					return nil
				}
			}
		}

		meta, annotation, _, err := parseNodeFile(s.fs, strings.TrimSuffix(path, ".md"))
		if err != nil {
			return nil //nolint:nilerr // skip unreadable nodes
		}

		nodeType := "note"
		if t, ok := meta["type"].(string); ok && t != "" {
			nodeType = t
		}
		if len(opts.Types) > 0 && !slices.Contains(opts.Types, nodeType) {
			return nil
		}

		manualOK := ManualProcessedEffective(meta)
		if opts.ManualProcessed != nil && manualOK != *opts.ManualProcessed {
			return nil
		}

		title := ""
		if t, ok := meta["title"].(string); ok && t != "" {
			title = t
		}
		if title == "" {
			parts := strings.Split(stemRel, "/")
			title = parts[len(parts)-1]
		}

		created := ""
		if c, ok := meta["created"].(string); ok {
			created = c
		}

		sourceURL := ""
		if u, ok := meta["source_url"].(string); ok {
			sourceURL = u
		}

		var translations []string
		switch v := meta["translations"].(type) {
		case []any:
			for _, item := range v {
				if str, ok := item.(string); ok {
					translations = append(translations, str)
				}
			}
		case []string:
			translations = append(translations, v...)
		}

		var keywords []string
		switch k := meta["keywords"].(type) {
		case []any:
			for _, item := range k {
				if str, ok := item.(string); ok {
					keywords = append(keywords, str)
				}
			}
		case []string:
			keywords = append(keywords, k...)
		}

		if opts.Q != "" {
			q := strings.ToLower(opts.Q)
			searchable := strings.ToLower(title) + " " + strings.ToLower(annotation)
			if kw, ok := meta["keywords"]; ok {
				switch k := kw.(type) {
				case []any:
					var searchableSb316 strings.Builder
					for _, item := range k {
						if str, ok := item.(string); ok {
							searchableSb316.WriteString(" " + strings.ToLower(str))
						}
					}
					searchable += searchableSb316.String()
				case []string:
					searchable += " " + strings.ToLower(strings.Join(k, " "))
				}
			}
			if !strings.Contains(searchable, q) {
				return nil
			}
		}

		items = append(items, &NodeListItem{
			Path:            stemRel,
			Title:           title,
			Type:            nodeType,
			Created:         created,
			SourceURL:       sourceURL,
			Translations:    translations,
			Annotation:      annotation,
			Keywords:        keywords,
			ManualProcessed: manualOK,
		})

		return nil
	})
	if err != nil {
		return nil, 0, errors.Errorf("list nodes: %w", err)
	}

	total := len(items)

	sort.Slice(items, func(i, j int) bool {
		var less bool
		switch sortField {
		case "title":
			less = items[i].Title < items[j].Title
		case "type":
			less = items[i].Type < items[j].Type
		case "source_url":
			less = items[i].SourceURL < items[j].SourceURL
		default:
			less = items[i].Created < items[j].Created
		}
		if order == "desc" {
			return !less
		}

		return less
	})

	if offset >= len(items) {
		return []*NodeListItem{}, total, nil
	}
	end := min(offset+limit, len(items))

	return items[offset:end], total, nil
}

// GetNode читает узел по пути стема (relative path от корня базы, без расширения .md).
func (s *Store) GetNode(ctx context.Context, basePath, nodePath string) (*Node, error) {
	basePath = filepath.Clean(basePath)
	stemPath := filepath.Join(basePath, filepath.FromSlash(nodePath))
	if !s.isNode(stemPath) {
		return nil, errors.Errorf("get node: %w", ErrNodeNotFound)
	}

	meta, annotation, content, err := parseNodeFile(s.fs, stemPath)
	if err != nil {
		return nil, errors.Errorf("get node: %w", err)
	}

	return &Node{
		Path:       filepath.ToSlash(nodePath),
		Annotation: annotation,
		Content:    content,
		Metadata:   NormalizeNodeMetadataForAPI(meta),
	}, nil
}

// PatchNodeManualProcessed устанавливает или снимает флаг manual_processed в frontmatter узла.
// При value=false ключ удаляется из YAML (семантика «не отмечено» как при отсутствии ключа).
func (s *Store) PatchNodeManualProcessed(ctx context.Context, basePath, nodePath string, value bool) error {
	_ = ctx
	basePath = filepath.Clean(basePath)
	stemPath := filepath.Join(basePath, filepath.FromSlash(nodePath))
	if !s.isNode(stemPath) {
		return errors.Errorf("patch node manual_processed: %w", ErrNodeNotFound)
	}
	matter, _, content, err := parseNodeFile(s.fs, stemPath)
	if err != nil {
		return errors.Errorf("patch node manual_processed: %w", err)
	}
	if value {
		matter["manual_processed"] = true
	} else {
		delete(matter, "manual_processed")
	}
	if msg := ValidateFrontmatter(matter); msg != "" {
		return errors.Errorf("patch node manual_processed: invalid frontmatter: %s", msg)
	}
	fmBytes, err := FormatFrontmatter(matter)
	if err != nil {
		return errors.Errorf("patch node manual_processed: %w", err)
	}
	var fileContent []byte
	fileContent = append(fileContent, fmBytes...)
	if content != "" {
		fileContent = append(fileContent, '\n')
		fileContent = append(fileContent, []byte(content)...)
		fileContent = append(fileContent, '\n')
	}
	mdPath := stemPath + ".md"
	tmpPath := mdPath + ".tmp"
	if err := afero.WriteFile(s.fs, tmpPath, fileContent, 0o644); err != nil {
		return errors.Errorf("patch node manual_processed: write: %w", err)
	}
	if err := s.fs.Rename(tmpPath, mdPath); err != nil {
		_ = s.fs.Remove(tmpPath)

		return errors.Errorf("patch node manual_processed: rename: %w", err)
	}

	return nil
}

// isNode проверяет, существует ли {stemPath}.md.
func (s *Store) isNode(stemPath string) bool {
	_, err := s.fs.Stat(stemPath + ".md")

	return err == nil
}

func (s *Store) validateTranslationFile(basePath, filePath, stemRel string, nodePaths map[string]struct{}, violations *[]ValidationError) error {
	data, err := afero.ReadFile(s.fs, filePath)
	if err != nil {
		*violations = append(*violations, ValidationError{Path: stemRel, Message: "cannot read translation file: " + err.Error()})

		return nil //nolint:nilerr
	}
	var matter map[string]any
	rest, err := frontmatter.Parse(strings.NewReader(string(data)), &matter)
	if err != nil {
		*violations = append(*violations, ValidationError{Path: stemRel, Message: "invalid frontmatter: " + err.Error()})

		return nil //nolint:nilerr
	}
	content := string(rest)

	translationOf, _ := matter["translation_of"].(string)
	if translationOf == "" {
		*violations = append(*violations, ValidationError{Path: stemRel, Message: "translation file: translation_of required"})

		return nil
	}
	lang, _ := matter["lang"].(string)
	if lang == "" {
		*violations = append(*violations, ValidationError{Path: stemRel, Message: "translation file: lang required"})

		return nil
	}

	// stemRel = "theme/slug.ru", extract themePath and baseSlug
	lastSlash := strings.LastIndex(stemRel, "/")
	var themePath, baseSlug string
	if lastSlash >= 0 {
		themePath = stemRel[:lastSlash]
		baseSlug = stemRel[lastSlash+1 : len(stemRel)-3] // remove ".ru"
	} else {
		baseSlug = stemRel[:len(stemRel)-3]
	}
	if translationOf != baseSlug {
		*violations = append(*violations, ValidationError{Path: stemRel, Message: "translation_of must match base slug"})

		return nil
	}

	originalPath := themePath
	if themePath != "" {
		originalPath += "/"
	}
	originalPath += baseSlug
	if _, ok := nodePaths[originalPath]; !ok {
		*violations = append(*violations, ValidationError{Path: stemRel, Message: "original node " + originalPath + " not found"})

		return nil
	}

	targets := ParseWikilinks(content)
	for _, target := range targets {
		if !s.wikilinkTargetExists(target, nodePaths) {
			*violations = append(*violations, ValidationError{Path: stemRel, Message: "wikilink target " + target + " not found"})
		}
	}

	// Check original has translations field
	originalStem := filepath.Join(basePath, filepath.FromSlash(originalPath))
	origMatter, _, _, parseErr := parseNodeFile(s.fs, originalStem)
	if parseErr != nil {
		*violations = append(*violations, ValidationError{Path: stemRel, Message: "cannot read original: " + parseErr.Error()})

		return nil //nolint:nilerr
	}
	translations := origMatter["translations"]
	var hasTranslation bool
	switch v := translations.(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s == baseSlug+".ru" {
				hasTranslation = true

				break
			}
		}
	case []string:
		if slices.Contains(v, baseSlug+".ru") {
			hasTranslation = true
		}
	}
	if !hasTranslation {
		*violations = append(*violations, ValidationError{Path: stemRel, Message: "original must have translations containing " + baseSlug + ".ru"})

		return nil
	}

	return nil
}

func (s *Store) wikilinkTargetExists(target string, nodePaths map[string]struct{}) bool {
	// Direct match
	if _, ok := nodePaths[target]; ok {
		return true
	}
	// Check themePath/target or any path ending with /target
	for path := range nodePaths {
		if path == target || strings.HasSuffix(path, "/"+target) {
			return true
		}
	}

	return false
}

func (s *Store) validateNode(filePath, stemRel string, violations *[]ValidationError) error {
	data, err := afero.ReadFile(s.fs, filePath)
	if err != nil {
		*violations = append(*violations, ValidationError{Path: stemRel, Message: "cannot read file: " + err.Error()})

		return nil //nolint:nilerr // violation recorded, continue walk
	}
	matter, err := parseFrontmatter(data)
	if err != nil {
		*violations = append(*violations, ValidationError{Path: stemRel, Message: "invalid frontmatter: " + err.Error()})

		return nil //nolint:nilerr // violation recorded, continue walk
	}
	if msg := ValidateFrontmatter(matter); msg != "" {
		*violations = append(*violations, ValidationError{Path: stemRel, Message: msg})
	}

	return nil
}

func (s *Store) readDir(path string) ([]os.FileInfo, error) {
	f, err := s.fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return f.Readdir(-1)
}

func (s *Store) buildTree(basePath, currentPath, relPath string, parent *TreeNode, depth int) error { //nolint:unparam // basePath passed to recursive calls
	if depth > maxDepth {
		return nil
	}
	entries, err := s.readDir(currentPath)
	if err != nil {
		return errors.Errorf("read dir %s: %w", currentPath, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		childPath := filepath.Join(currentPath, name)
		childRel := filepath.Join(relPath, name)
		// Пропускаем директории вложений: если рядом существует {name}.md.
		if _, err := s.fs.Stat(filepath.Join(currentPath, name+".md")); err == nil {
			continue
		}
		child := &TreeNode{
			Name: name,
			Path: filepath.ToSlash(childRel),
		}
		if err := s.buildTree(basePath, childPath, childRel, child, depth+1); err != nil {
			return err
		}
		parent.Children = append(parent.Children, child)
	}

	return nil
}
