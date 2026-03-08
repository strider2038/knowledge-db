import * as React from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import rehypeSlug from 'rehype-slug'
import { MermaidDiagram } from '@/components/MermaidDiagram'
import '@/lib/highlight'

interface MarkdownContentProps {
  content: string
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

export function MarkdownContent({ content }: MarkdownContentProps) {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      rehypePlugins={[rehypeSlug, rehypeHighlight]}
      components={{
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
          return <pre {...props}>{children}</pre>
        },
      }}
    >
      {content}
    </ReactMarkdown>
  )
}
