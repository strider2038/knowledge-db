package api_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muonsoft/api-testing/apitest"
	"github.com/muonsoft/api-testing/assertjson"
	"github.com/stretchr/testify/require"
	"github.com/strider2038/knowledge-db/internal/api"
	"github.com/strider2038/knowledge-db/internal/ingestion"
)

func setupTestHandlerWithAnnotationsNode(t *testing.T) http.Handler {
	t.Helper()
	tmp := t.TempDir()
	themeDir := filepath.Join(tmp, "topic")
	require.NoError(t, os.MkdirAll(themeDir, 0o755))
	content := injectTestNodeID(`---
keywords: [test]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
type: note
title: My Node
---

exactly-once delivery`, "topic/my-node.md")
	require.NoError(t, os.WriteFile(filepath.Join(themeDir, "my-node.md"), []byte(content), 0o644))
	h := api.NewHandler(tmp, &ingestion.StubIngester{})
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	return mux
}

func TestListNodeAnnotations_WhenEmpty_ExpectOK(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerWithAnnotationsNode(t)

	resp := apitest.HandleGET(t, handler, "/api/nodes/topic/my-node/annotations")

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("notes").IsArray().WithLength(0)
	})
}

func TestCreateNodeAnnotation_WhenGeneral_Expect201(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerWithAnnotationsNode(t)

	resp := apitest.HandlePOST(t, handler, "/api/nodes/topic/my-node/annotations",
		strings.NewReader(`{"body":"My note"}`),
		apitest.WithJSONContentType())

	resp.IsCreated()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("id").IsString().IsNotEmpty()
		json.Node("body").IsString().EqualTo("My note")
		json.Node("anchor").IsNull()
	})
}

func TestCreateNodeAnnotation_WhenAnchored_ExpectResolved(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerWithAnnotationsNode(t)

	resp := apitest.HandlePOST(t, handler, "/api/nodes/topic/my-node/annotations",
		strings.NewReader(`{"body":"check","anchor":{"type":"text_quote","content_path":"topic/my-node","exact":"exactly-once delivery"}}`),
		apitest.WithJSONContentType())

	resp.IsCreated()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("resolved").IsTrue()
	})
}

func TestNodeAnnotations_WhenTranslationPath_ExpectSameList(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerWithAnnotationsNode(t)
	apitest.HandlePOST(t, handler, "/api/nodes/topic/my-node/annotations",
		strings.NewReader(`{"body":"shared"}`),
		apitest.WithJSONContentType()).IsCreated()

	resp := apitest.HandleGET(t, handler, "/api/nodes/topic/my-node.ru/annotations")

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("notes").IsArray().WithLength(1)
		json.Node("notes", 0, "body").IsString().EqualTo("shared")
	})
}

func TestUpdateNodeAnnotation_WhenExists_ExpectOK(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerWithAnnotationsNode(t)
	apitest.HandlePOST(t, handler, "/api/nodes/topic/my-node/annotations",
		strings.NewReader(`{"body":"v1"}`),
		apitest.WithJSONContentType()).IsCreated()

	list := apitest.HandleGET(t, handler, "/api/nodes/topic/my-node/annotations")
	var noteID string
	list.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("notes", 0, "id").IsString().IsNotEmpty()
		noteID = json.Node("notes", 0, "id").String()
	})

	resp := apitest.HandlePATCH(t, handler, "/api/nodes/topic/my-node/annotations/"+noteID,
		strings.NewReader(`{"body":"v2"}`),
		apitest.WithJSONContentType())

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("body").IsString().EqualTo("v2")
	})
}

func TestDeleteNodeAnnotation_WhenExists_ExpectOK(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerWithAnnotationsNode(t)
	apitest.HandlePOST(t, handler, "/api/nodes/topic/my-node/annotations",
		strings.NewReader(`{"body":"v1"}`),
		apitest.WithJSONContentType()).IsCreated()

	list := apitest.HandleGET(t, handler, "/api/nodes/topic/my-node/annotations")
	var noteID string
	list.HasJSON(func(json *assertjson.AssertJSON) {
		noteID = json.Node("notes", 0, "id").String()
	})

	resp := apitest.HandleDELETE(t, handler, "/api/nodes/topic/my-node/annotations/"+noteID)

	resp.IsOK()
	list2 := apitest.HandleGET(t, handler, "/api/nodes/topic/my-node/annotations")
	list2.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("notes").IsArray().WithLength(0)
	})
}

func TestCreateNodeAnnotation_WhenInvalidBody_Expect400(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerWithAnnotationsNode(t)

	resp := apitest.HandlePOST(t, handler, "/api/nodes/topic/my-node/annotations",
		strings.NewReader(`{"body":""}`),
		apitest.WithJSONContentType())

	resp.IsBadRequest()
}
