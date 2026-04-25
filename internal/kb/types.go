package kb

// TreeNode — узел дерева тем (тема или подтема).
type TreeNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	Children []*TreeNode `json:"children,omitempty"`
}

// Node — узел базы знаний (папка со статьёй).
type Node struct {
	Path       string         `json:"path"`
	Annotation string         `json:"annotation"`
	Content    string         `json:"content"`
	Metadata   map[string]any `json:"metadata"`
}

// NodeListItem — элемент списка узлов для обзора (метаданные без content).
type NodeListItem struct {
	Path            string   `json:"path"`
	Title           string   `json:"title"`
	Type            string   `json:"type"`
	Created         string   `json:"created"`
	SourceURL       string   `json:"source_url"` //nolint:tagliatelle // REST API snake_case
	Translations    []string `json:"translations,omitempty"`
	Annotation      string   `json:"annotation,omitempty"`
	Keywords        []string `json:"keywords,omitempty"`
	ManualProcessed bool     `json:"manual_processed"` //nolint:tagliatelle // REST API snake_case
}

// ListNodesOptions — параметры для ListNodesWithOptions.
type ListNodesOptions struct {
	Path      string   // путь темы (пустой = вся база)
	Recursive bool     // true = узлы всего поддерева
	Q         string   // подстрока поиска в title, keywords, annotation
	Types     []string // фильтр по типу: article, link, note
	Sort      string   // title, type, created, source_url
	Order     string   // asc, desc
	Limit     int      // лимит строк (default 50, max 200)
	Offset    int      // смещение
	// ManualProcessed — если не nil, только узлы с совпадающим флагом (GET /api/nodes?manual_processed=).
	ManualProcessed *bool
}

// ValidationError — ошибка валидации с путём.
type ValidationError struct {
	Path    string
	Message string
}

func (e ValidationError) Error() string {
	if e.Path != "" {
		return e.Path + ": " + e.Message
	}

	return e.Message
}
