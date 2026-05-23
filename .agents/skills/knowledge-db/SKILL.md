---
name: knowledge-db
description: Knowledge base layout and node format for knowledge-db. Use when creating or editing KB markdown files. Root path placeholder {{DATA_PATH}}.
---

# Knowledge base — knowledge-db

Use this skill when working with the user's knowledge base files (not application source).

## Base path

Knowledge base root: **{{DATA_PATH}}**

In the running app, this is `KB_DATA_PATH`. Do not assume `./data` unless the user specifies it.

## Node structure

Each node (article, link, note) is a **directory** with a main markdown file `{folder-name}.md` (Obsidian-style):

| Path | Purpose |
|------|---------|
| `{dirname}/{dirname}.md` | Main file: YAML frontmatter + markdown body |
| `notes/` | Extra notes (`.md`) |
| `images/` | Images |
| `.local/` | Local-only data (gitignored) |

## Main `.md` frontmatter

```yaml
---
keywords: [tag1, tag2]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
annotation: "Краткое описание"   # optional
source: "https://..."            # optional
sourceType: article              # optional — article | link | note
---

# Title

Body (markdown)…
```

Required frontmatter fields: `keywords`, `created`, `updated`.

User-facing content is often **Russian**; keep existing language unless asked to translate.

## Topic hierarchy

- Topics/subtopics are directories (typically 2–3 levels)
- Nodes live under topics as folders with `{dirname}.md`

## Creating a node

1. Create folder under the target topic, e.g. `topic/subtopic/my-node/`
2. Add `my-node.md` with frontmatter and body
3. Add `notes/` or `images/` if needed

## Validation

Run from the app repo:

```bash
go run ./cmd/kb-cli validate --path "{{DATA_PATH}}"
```

Follow OpenSpec and `.cursor/rules/markdown-normalization.mdc` when normalizing imported content.
