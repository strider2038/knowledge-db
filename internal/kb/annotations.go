package kb

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/muonsoft/errors"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

const (
	annotationsFileName      = "annotations.yaml"
	annotationsFileVersion   = 1
	maxAnnotationsPerNode    = 200
	maxAnnotationBodyLength  = 4000
	maxAnnotationExactLength = 500
	maxAnnotationContextLen  = 80
	anchorTypeTextQuote      = "text_quote"
)

// ErrAnnotationNotFound — аннотация не найдена в sidecar.
var ErrAnnotationNotFound = errors.New("annotation not found")

// ErrInvalidAnnotation — невалидные данные аннотации.
var ErrInvalidAnnotation = errors.New("invalid annotation")

// AnnotationAnchor describes a text fragment anchor.
type AnnotationAnchor struct {
	Type        string `json:"type" yaml:"type"`
	ContentPath string `json:"content_path" yaml:"content_path"` //nolint:tagliatelle // sidecar YAML uses snake_case keys
	Exact       string `json:"exact" yaml:"exact"`
	Prefix      string `json:"prefix,omitempty" yaml:"prefix,omitempty"`
	Suffix      string `json:"suffix,omitempty" yaml:"suffix,omitempty"`
	HeadingID   string `json:"heading_id,omitempty" yaml:"heading_id,omitempty"` //nolint:tagliatelle // sidecar YAML uses snake_case keys
}

// NodeAnnotation is a stored personal annotation.
type NodeAnnotation struct {
	ID       string            `json:"id" yaml:"id"`
	Created  string            `json:"created" yaml:"created"`
	Updated  string            `json:"updated" yaml:"updated"`
	Body     string            `json:"body" yaml:"body"`
	Anchor   *AnnotationAnchor `json:"anchor" yaml:"anchor"`
	Resolved *bool             `json:"resolved,omitempty" yaml:"-"`
}

type annotationsFile struct {
	Version int              `yaml:"version"`
	Notes   []nodeAnnotation `yaml:"notes"`
}

type nodeAnnotation struct {
	ID      string            `yaml:"id"`
	Created string            `yaml:"created"`
	Updated string            `yaml:"updated"`
	Body    string            `yaml:"body"`
	Anchor  *AnnotationAnchor `yaml:"anchor"`
}

// CreateAnnotationParams holds input for a new annotation.
type CreateAnnotationParams struct {
	Body   string
	Anchor *AnnotationAnchor
}

// UpdateAnnotationParams holds partial update fields.
type UpdateAnnotationParams struct {
	Body   *string
	Anchor *AnnotationAnchor
}

// AnnotationsBaseNodePath returns the base node path without translation suffix.
func AnnotationsBaseNodePath(nodePath string) string {
	nodePath = filepath.ToSlash(filepath.Clean(filepath.FromSlash(nodePath)))
	if nodePath == "." || nodePath == "" {
		return nodePath
	}
	parts := strings.Split(nodePath, "/")
	last := parts[len(parts)-1]
	if dot := strings.LastIndex(last, "."); dot > 0 {
		suffix := last[dot+1:]
		if len(suffix) == 2 && isLowerASCIILetters(suffix) {
			parts[len(parts)-1] = last[:dot]

			return strings.Join(parts, "/")
		}
	}

	return nodePath
}

func isLowerASCIILetters(s string) bool {
	for i := range len(s) {
		if s[i] < 'a' || s[i] > 'z' {
			return false
		}
	}

	return true
}

func (s *Store) annotationsFilePath(basePath, nodePath string) string {
	baseNode := AnnotationsBaseNodePath(nodePath)
	stem := filepath.Join(basePath, filepath.FromSlash(baseNode))

	return filepath.Join(stem, annotationsFileName)
}

func (s *Store) ensureBaseNodeExists(basePath, nodePath string) error {
	baseNode := AnnotationsBaseNodePath(nodePath)
	stem := filepath.Join(basePath, filepath.FromSlash(baseNode))
	if !s.isNode(stem) {
		return errors.Errorf("annotations: %w", ErrNodeNotFound)
	}

	return nil
}

// ListNodeAnnotations returns annotations for the logical node.
func (s *Store) ListNodeAnnotations(ctx context.Context, basePath, nodePath string) ([]NodeAnnotation, error) {
	_ = ctx
	if err := s.ensureBaseNodeExists(basePath, nodePath); err != nil {
		return nil, err
	}
	file, err := s.readAnnotationsFile(basePath, nodePath)
	if err != nil {
		return nil, err
	}
	out := make([]NodeAnnotation, 0, len(file.Notes))
	bodyCache := map[string]string{}
	for _, note := range file.Notes {
		ann := note.toAPI()
		s.attachResolved(basePath, &ann, bodyCache)
		out = append(out, ann)
	}

	return out, nil
}

// CreateNodeAnnotation appends a new annotation.
func (s *Store) CreateNodeAnnotation(ctx context.Context, basePath, nodePath string, params CreateAnnotationParams) (NodeAnnotation, error) {
	_ = ctx
	if err := s.ensureBaseNodeExists(basePath, nodePath); err != nil {
		return NodeAnnotation{}, err
	}
	if err := validateAnnotationBody(params.Body); err != nil {
		return NodeAnnotation{}, err
	}
	anchor, err := normalizeAnchor(params.Anchor)
	if err != nil {
		return NodeAnnotation{}, err
	}
	file, err := s.readAnnotationsFile(basePath, nodePath)
	if err != nil {
		return NodeAnnotation{}, err
	}
	if len(file.Notes) >= maxAnnotationsPerNode {
		return NodeAnnotation{}, errors.Errorf("create annotation: %w: too many annotations", ErrInvalidAnnotation)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	id, err := NewNodeID()
	if err != nil {
		return NodeAnnotation{}, errors.Errorf("create annotation: %w", err)
	}
	note := nodeAnnotation{
		ID:      id,
		Created: now,
		Updated: now,
		Body:    params.Body,
		Anchor:  anchor,
	}
	file.Notes = append(file.Notes, note)
	if err := s.writeAnnotationsFile(basePath, nodePath, file); err != nil {
		return NodeAnnotation{}, err
	}
	ann := note.toAPI()
	s.attachResolved(basePath, &ann, map[string]string{})

	return ann, nil
}

// UpdateNodeAnnotation updates an existing annotation.
func (s *Store) UpdateNodeAnnotation(ctx context.Context, basePath, nodePath, id string, params UpdateAnnotationParams) (NodeAnnotation, error) {
	_ = ctx
	if err := s.ensureBaseNodeExists(basePath, nodePath); err != nil {
		return NodeAnnotation{}, err
	}
	file, err := s.readAnnotationsFile(basePath, nodePath)
	if err != nil {
		return NodeAnnotation{}, err
	}
	idx := findAnnotationIndex(file.Notes, id)
	if idx < 0 {
		return NodeAnnotation{}, errors.Errorf("update annotation: %w", ErrAnnotationNotFound)
	}
	note := file.Notes[idx]
	if params.Body != nil {
		if err := validateAnnotationBody(*params.Body); err != nil {
			return NodeAnnotation{}, err
		}
		note.Body = *params.Body
	}
	if params.Anchor != nil {
		anchor, err := normalizeAnchor(params.Anchor)
		if err != nil {
			return NodeAnnotation{}, err
		}
		note.Anchor = anchor
	}
	note.Updated = time.Now().UTC().Format(time.RFC3339)
	file.Notes[idx] = note
	if err := s.writeAnnotationsFile(basePath, nodePath, file); err != nil {
		return NodeAnnotation{}, err
	}
	ann := note.toAPI()
	s.attachResolved(basePath, &ann, map[string]string{})

	return ann, nil
}

// DeleteNodeAnnotation removes an annotation by id.
func (s *Store) DeleteNodeAnnotation(ctx context.Context, basePath, nodePath, id string) error {
	_ = ctx
	if err := s.ensureBaseNodeExists(basePath, nodePath); err != nil {
		return err
	}
	file, err := s.readAnnotationsFile(basePath, nodePath)
	if err != nil {
		return err
	}
	idx := findAnnotationIndex(file.Notes, id)
	if idx < 0 {
		return errors.Errorf("delete annotation: %w", ErrAnnotationNotFound)
	}
	file.Notes = append(file.Notes[:idx], file.Notes[idx+1:]...)
	if len(file.Notes) == 0 {
		return s.removeAnnotationsFile(basePath, nodePath)
	}

	return s.writeAnnotationsFile(basePath, nodePath, file)
}

func (n nodeAnnotation) toAPI() NodeAnnotation {
	return NodeAnnotation{
		ID:      n.ID,
		Created: n.Created,
		Updated: n.Updated,
		Body:    n.Body,
		Anchor:  cloneAnchor(n.Anchor),
	}
}

func cloneAnchor(a *AnnotationAnchor) *AnnotationAnchor {
	if a == nil {
		return nil
	}
	cp := *a

	return &cp
}

func findAnnotationIndex(notes []nodeAnnotation, id string) int {
	for i, note := range notes {
		if note.ID == id {
			return i
		}
	}

	return -1
}

func (s *Store) readAnnotationsFile(basePath, nodePath string) (annotationsFile, error) {
	path := s.annotationsFilePath(basePath, nodePath)
	data, err := afero.ReadFile(s.fs, path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return annotationsFile{Version: annotationsFileVersion, Notes: []nodeAnnotation{}}, nil
		}

		return annotationsFile{}, errors.Errorf("read annotations: %w", err)
	}
	var file annotationsFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return annotationsFile{}, errors.Errorf("read annotations: parse yaml: %w", err)
	}
	if file.Version == 0 {
		file.Version = annotationsFileVersion
	}
	if file.Notes == nil {
		file.Notes = []nodeAnnotation{}
	}

	return file, nil
}

func (s *Store) writeAnnotationsFile(basePath, nodePath string, file annotationsFile) error {
	path := s.annotationsFilePath(basePath, nodePath)
	if err := s.fs.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errors.Errorf("write annotations: mkdir: %w", err)
	}
	file.Version = annotationsFileVersion
	data, err := yaml.Marshal(file)
	if err != nil {
		return errors.Errorf("write annotations: marshal yaml: %w", err)
	}
	if err := afero.WriteFile(s.fs, path, data, 0o644); err != nil {
		return errors.Errorf("write annotations: %w", err)
	}

	return nil
}

func (s *Store) removeAnnotationsFile(basePath, nodePath string) error {
	path := s.annotationsFilePath(basePath, nodePath)
	if err := s.fs.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return errors.Errorf("remove annotations: %w", err)
	}

	return nil
}

func validateAnnotationBody(body string) error {
	if strings.TrimSpace(body) == "" {
		return errors.Errorf("validate annotation: %w: body is required", ErrInvalidAnnotation)
	}
	if len([]rune(body)) > maxAnnotationBodyLength {
		return errors.Errorf("validate annotation: %w: body too long", ErrInvalidAnnotation)
	}
	for _, r := range body {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return errors.Errorf("validate annotation: %w: control characters not allowed", ErrInvalidAnnotation)
		}
	}

	return nil
}

func normalizeAnchor(anchor *AnnotationAnchor) (*AnnotationAnchor, error) {
	if anchor == nil {
		return nil, nil //nolint:nilnil // general notes have no anchor
	}
	if anchor.Type != "" && anchor.Type != anchorTypeTextQuote {
		return nil, errors.Errorf("normalize anchor: %w: unsupported anchor type", ErrInvalidAnnotation)
	}
	contentPath := filepath.ToSlash(strings.TrimSpace(anchor.ContentPath))
	exact := strings.TrimSpace(anchor.Exact)
	if contentPath == "" || exact == "" {
		return nil, errors.Errorf("normalize anchor: %w: content_path and exact are required", ErrInvalidAnnotation)
	}
	if len([]rune(exact)) > maxAnnotationExactLength {
		return nil, errors.Errorf("normalize anchor: %w: exact too long", ErrInvalidAnnotation)
	}
	prefix := trimContext(anchor.Prefix)
	suffix := trimContext(anchor.Suffix)
	headingID := strings.TrimSpace(anchor.HeadingID)

	return &AnnotationAnchor{
		Type:        anchorTypeTextQuote,
		ContentPath: contentPath,
		Exact:       exact,
		Prefix:      prefix,
		Suffix:      suffix,
		HeadingID:   headingID,
	}, nil
}

func trimContext(s string) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) > maxAnnotationContextLen {
		return string(runes[len(runes)-maxAnnotationContextLen:])
	}

	return s
}

func (s *Store) attachResolved(basePath string, ann *NodeAnnotation, bodyCache map[string]string) {
	if ann == nil || ann.Anchor == nil {
		return
	}
	contentPath := ann.Anchor.ContentPath
	content, ok := bodyCache[contentPath]
	if !ok {
		var err error
		content, err = s.nodeBodyForPath(basePath, contentPath)
		if err != nil {
			resolved := false
			ann.Resolved = &resolved

			return
		}
		bodyCache[contentPath] = content
	}
	resolved := resolveTextQuote(content, ann.Anchor)
	ann.Resolved = &resolved
}

func (s *Store) nodeBodyForPath(basePath, nodePath string) (string, error) {
	stem := filepath.Join(basePath, filepath.FromSlash(nodePath))
	meta, _, content, err := parseNodeFile(s.fs, stem)
	if err != nil {
		return "", errors.Errorf("node body: %w", err)
	}
	_ = meta

	return content, nil
}
