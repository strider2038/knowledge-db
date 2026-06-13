package api

import "strings"

const annotationsPathSuffix = "/annotations"

// nodeGETKind classifies GET /api/nodes/{path...} targets.
type nodeGETKind int

const (
	nodeGETKindNode nodeGETKind = iota
	nodeGETKindAnnotations
)

func classifyNodeGET(path string) (nodeGETKind, string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nodeGETKindNode, "", false
	}
	if nodePath, ok := annotationsListPath(path); ok {
		return nodeGETKindAnnotations, nodePath, true
	}

	return nodeGETKindNode, path, true
}

func classifyNodeDELETE(path string) (nodeDELETEKind, string, string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nodeDELETEKindNode, "", "", false
	}
	if nodePath, noteID, ok := annotationsItemPath(path); ok {
		return nodeDELETEKindAnnotation, nodePath, noteID, true
	}

	return nodeDELETEKindNode, path, "", true
}

type nodeDELETEKind int

const (
	nodeDELETEKindNode nodeDELETEKind = iota
	nodeDELETEKindAnnotation
)

func classifyNodePATCH(path string) (nodePATCHKind, string, string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nodePATCHKindNode, "", "", false
	}
	if nodePath, noteID, ok := annotationsItemPath(path); ok {
		return nodePATCHKindAnnotation, nodePath, noteID, true
	}

	return nodePATCHKindNode, path, "", true
}

type nodePATCHKind int

const (
	nodePATCHKindNode nodePATCHKind = iota
	nodePATCHKindAnnotation
)

func classifyNodePOST(path string) (nodePOSTKind, string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nodePOSTKindMove, "", false
	}
	if nodePath, ok := annotationsListPath(path); ok {
		return nodePOSTKindAnnotation, nodePath, true
	}

	return nodePOSTKindMove, path, true
}

type nodePOSTKind int

const (
	nodePOSTKindMove nodePOSTKind = iota
	nodePOSTKindAnnotation
)

func annotationsListPath(path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasSuffix(path, annotationsPathSuffix) {
		return "", false
	}
	nodePath := strings.TrimSuffix(path, annotationsPathSuffix)
	if nodePath == "" {
		return "", false
	}

	return nodePath, true
}

func annotationsItemPath(path string) (string, string, bool) {
	path = strings.TrimSpace(path)
	before, noteID, found := strings.Cut(path, annotationsPathSuffix+"/")
	if !found {
		return "", "", false
	}
	nodePath := before
	noteID = strings.TrimSpace(noteID)
	if nodePath == "" || noteID == "" {
		return "", "", false
	}

	return nodePath, noteID, true
}
