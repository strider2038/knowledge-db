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
