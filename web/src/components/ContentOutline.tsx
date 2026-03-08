import { useState } from 'react'
import { List, X } from 'lucide-react'
import { extractHeadings } from '@/lib/headings'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'

interface ContentOutlineProps {
  content: string
  className?: string
}

function scrollToHeading(slug: string) {
  const el = document.getElementById(slug)
  el?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

interface HeadingNode {
  heading: { level: number; text: string; slug: string }
  children: HeadingNode[]
}

function buildHeadingTree(headings: { level: number; text: string; slug: string }[]): HeadingNode[] {
  const stack: { level: number; node: HeadingNode }[] = []
  const root: HeadingNode[] = []

  for (const h of headings) {
    const node: HeadingNode = { heading: h, children: [] }
    while (stack.length > 0 && stack[stack.length - 1].level >= h.level) {
      stack.pop()
    }
    if (stack.length === 0) {
      root.push(node)
    } else {
      stack[stack.length - 1].node.children.push(node)
    }
    stack.push({ level: h.level, node })
  }
  return root
}

function OutlineTree({
  nodes,
  depth,
  onSelect,
}: {
  nodes: HeadingNode[]
  depth: number
  onSelect?: () => void
}) {
  const isRoot = depth === 0
  return (
    <ul
      className={cn(
        'list-none space-y-0',
        !isRoot && 'ml-4 border-l border-border pl-2'
      )}
    >
      {nodes.map(({ heading, children }) => (
        <li key={heading.slug}>
          <button
            type="button"
            onClick={() => {
              scrollToHeading(heading.slug)
              onSelect?.()
            }}
            className="block w-full rounded px-2 py-1 text-left text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          >
            {heading.text}
          </button>
          {children.length > 0 && (
            <OutlineTree nodes={children} depth={depth + 1} onSelect={onSelect} />
          )}
        </li>
      ))}
    </ul>
  )
}

function OutlineList({
  headings,
  onSelect,
}: {
  headings: { level: number; text: string; slug: string }[]
  onSelect?: () => void
}) {
  const tree = buildHeadingTree(headings)
  return <OutlineTree nodes={tree} depth={0} onSelect={onSelect} />
}

export function ContentOutline({ content, className }: ContentOutlineProps) {
  const headings = extractHeadings(content)
  if (headings.length === 0) return null

  return (
    <nav
      className={cn(
        'sticky top-4 flex min-h-[calc(100vh-1rem)] max-h-[calc(100vh-1rem)] flex-col overflow-hidden text-sm',
        className
      )}
      aria-label="Содержание"
    >
      <div className="mb-1.5 shrink-0 font-semibold text-foreground">Содержание</div>
      <div className="min-h-0 overflow-y-auto">
        <OutlineList headings={headings} />
      </div>
    </nav>
  )
}

export function ContentOutlineFloating({ content }: { content: string }) {
  const [open, setOpen] = useState(false)
  const headings = extractHeadings(content)
  if (headings.length === 0) return null

  return (
    <div className="fixed bottom-6 left-6 z-50 lg:hidden">
      <Button
        variant="outline"
        size="icon"
        className="size-10 rounded-full shadow-lg"
        onClick={() => setOpen((o) => !o)}
        aria-label="Содержание"
        aria-expanded={open}
      >
        {open ? <X className="size-5" /> : <List className="size-5" />}
      </Button>
      {open && (
        <div className="absolute bottom-14 left-0 flex max-h-[70vh] w-64 flex-col overflow-hidden rounded-lg border bg-popover p-3 shadow-lg">
          <div className="mb-1.5 shrink-0 font-semibold text-foreground">Содержание</div>
          <div className="min-h-0 overflow-y-auto">
            <OutlineList headings={headings} onSelect={() => setOpen(false)} />
          </div>
        </div>
      )}
    </div>
  )
}
