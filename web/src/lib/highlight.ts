/**
 * highlight.js через lowlight (rehype-highlight).
 * По умолчанию rehype-highlight берёт только `lowlight/common` — Dockerfile туда не входит,
 * поэтому явно добавляем grammar и алиас `docker` (как в highlight.js).
 * Темы: github (light) + github-dark (.dark) в highlight.css.
 */
import '@/styles/highlight.css'

import dockerfile from 'highlight.js/lib/languages/dockerfile'
import { common } from 'lowlight'

/** Расширенный набор языков для rehype-highlight (common + Dockerfile). */
export const rehypeHighlightLanguages = {
  ...common,
  dockerfile,
}

/** Ограждения ```docker → dockerfile (hljs aliases). */
export const rehypeHighlightAliases = {
  docker: 'dockerfile',
}
