/**
 * highlight.js setup for MarkdownContent.
 * Стили и явный импорт нужных языков (task 1.2):
 * javascript, typescript, bash, json, yaml, python, go.
 * rehype-highlight по умолчанию использует lowlight/common с этими языками.
 * Темы: github (light) + github-dark (.dark) для читаемости.
 */
import '@/styles/highlight.css'

// Явный импорт языков для подсветки кода (уменьшает bundle при tree-shaking)
import 'highlight.js/lib/languages/javascript'
import 'highlight.js/lib/languages/typescript'
import 'highlight.js/lib/languages/bash'
import 'highlight.js/lib/languages/json'
import 'highlight.js/lib/languages/yaml'
import 'highlight.js/lib/languages/python'
import 'highlight.js/lib/languages/go'
