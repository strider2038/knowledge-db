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

interface MarkdownContentProps {
  content: string
  /** Путь узла (theme/slug) для разрешения относительных путей изображений. */
  nodePath?: string
}

function isMermaidCodeBlock(
  children: React.ReactNode
): { code: string } | null {
  const child = React.Children.only(children) as
    | React.ReactElement<{ className?: string; children?: React.ReactNode }>
    | undefined
  if (!child || child.type !== 'code') return null
  const className = child.props?.className
  if (!className?.includes('language-mermaid')) return null
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

export function MarkdownContent({ content, nodePath }: MarkdownContentProps) {
  return (
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
      }}
    >
      {content}
    </ReactMarkdown>
  )
}
