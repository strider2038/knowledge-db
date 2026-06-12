import * as React from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import rehypeSlug from 'rehype-slug'
import { MermaidDiagram } from '@/components/MermaidDiagram'
import { CodeBlock } from '@/components/CodeBlock'
import { getAssetUrl } from '@/services/api'
import {
  rehypeHighlightAliases,
  rehypeHighlightLanguages,
} from '@/lib/highlight'
import { cn } from '@/lib/utils'
import { flattenMarkdownText } from '@/lib/markdown-text'

interface MarkdownContentProps {
  content: string
  /**
   * Путь узла (theme/slug) для разрешения относительных путей изображений.
   * Для переводов (`slug.ru`) передавай путь к оригиналу без суффикса языка,
   * иначе картинки из папки `{slug}/images/` не найдутся.
   */
  nodePath?: string
  className?: string
  paragraphClassName?: (text: string) => string | undefined
  paragraphPrefix?: (text: string) => React.ReactNode
}

function classNameToString(className: unknown): string {
  if (typeof className === 'string') return className
  if (Array.isArray(className)) return className.filter(Boolean).join(' ')
  if (className != null) return String(className)
  return ''
}

function isMermaidCodeBlock(
  children: React.ReactNode
): { code: string } | null {
  const child = React.Children.only(children) as
    | React.ReactElement<{ className?: unknown; children?: React.ReactNode }>
    | undefined
  if (!child || !React.isValidElement(child)) return null
  const className = classNameToString(child.props?.className)
  if (!className.includes('language-mermaid')) return null
  const code = child.props?.children
  const codeStr =
    typeof code === 'string' ? code : Array.isArray(code) ? code.join('') : ''
  return { code: codeStr }
}

function resolveImageSrc(src: string | undefined, nodePath: string | undefined): string | undefined {
  if (!src || !nodePath) return src
  if (src.startsWith('http://') || src.startsWith('https://') || src.startsWith('/')) return src
  const imagesIdx = src.indexOf('images/')
  const assetPath =
    imagesIdx >= 0
      ? `${nodePath}/images/${src.slice(imagesIdx + 7)}`
      : `${nodePath}/${src.replace(/^\.\//, '')}`
  return getAssetUrl(assetPath)
}

/** Plain text from markdown <code> children (strings or nested spans from highlight.js). */
function flattenCodeText(node: React.ReactNode): string {
  return flattenMarkdownText(node)
}

function isBlockMarkdownCode(className: unknown, children: React.ReactNode): boolean {
  const cls = classNameToString(className)
  if (/\blanguage-[\w-]+\b/.test(cls) || /\bhljs\b/.test(cls)) return true
  // Fenced block without language: mdast appends trailing '\n' to code text; inline collapses newlines to spaces.
  return flattenCodeText(children).endsWith('\n')
}

export function MarkdownContent({
  content,
  nodePath,
  className,
  paragraphClassName,
  paragraphPrefix,
}: MarkdownContentProps) {
  return (
    <div className={className}>
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      rehypePlugins={[
        rehypeSlug,
        [
          rehypeHighlight,
          {
            languages: rehypeHighlightLanguages,
            aliases: rehypeHighlightAliases,
          },
        ],
      ]}
      components={{
        img: ({ src, alt, ...props }) => (
          <img {...props} src={resolveImageSrc(src, nodePath)} alt={alt ?? ''} />
        ),
        a: ({ href, children, ...props }) => (
          <a
            {...props}
            href={href}
            target="_blank"
            rel="noopener noreferrer"
          >
            {children}
          </a>
        ),
        pre: ({ children, ...props }) => {
          const mermaidData = isMermaidCodeBlock(children)
          if (mermaidData) {
            return <MermaidDiagram code={mermaidData.code} />
          }
          return <CodeBlock preProps={props}>{children}</CodeBlock>
        },
        table: ({ children, className, ...props }) => (
          <div className="my-4 w-full min-w-0 overflow-x-auto overscroll-x-contain">
            <table {...props} className={cn(className)}>
              {children}
            </table>
          </div>
        ),
        code: ({ children, className, node, ...props }) => {
          void node
          if (isBlockMarkdownCode(className, children)) {
            return (
              <code {...props} className={cn(className)}>
                {children}
              </code>
            )
          }
          return (
            <code
              {...props}
              className={cn(
                'rounded-md border border-border/70 bg-muted px-[0.35em] py-[0.12em] font-mono text-[0.875em] font-medium leading-snug text-foreground [font-variant-ligatures:none] before:content-none after:content-none break-words',
                className
              )}
            >
              {children}
            </code>
          )
        },
        p: ({ children, className, ...props }) => {
          const text = flattenMarkdownText(children)
          const extra = paragraphClassName?.(text)
          const prefix = paragraphPrefix?.(text)
          return (
            <p {...props} className={cn(className, extra)}>
              {prefix}
              {children}
            </p>
          )
        },
      }}
    >
      {content}
    </ReactMarkdown>
    </div>
  )
}
