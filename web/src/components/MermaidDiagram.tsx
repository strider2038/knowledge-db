import { useId, useLayoutEffect, useRef, useState } from 'react'
import { useTheme } from 'next-themes'
import mermaid from 'mermaid'

let lastMermaidTheme: 'light' | 'dark' | null = null

function initMermaid(theme: 'light' | 'dark') {
  if (lastMermaidTheme === theme) return
  mermaid.initialize({
    startOnLoad: false,
    theme: theme === 'dark' ? 'dark' : 'default',
  })
  lastMermaidTheme = theme
}

interface MermaidDiagramProps {
  code: string
}

export function MermaidDiagram({ code }: MermaidDiagramProps) {
  const id = `mermaid-${useId().replace(/:/g, '-')}`
  const containerRef = useRef<HTMLDivElement>(null)
  const [error, setError] = useState<string | null>(null)
  const { resolvedTheme } = useTheme()

  useLayoutEffect(() => {
    const theme = resolvedTheme === 'dark' ? 'dark' : 'light'
    initMermaid(theme)

    const container = containerRef.current
    if (!container) return

    let active = true

    mermaid
      .render(id, code.trim())
      .then(({ svg, bindFunctions }) => {
        if (!active || !container) return
        container.innerHTML = svg
        if (bindFunctions) {
          bindFunctions(container)
        }
        setError(null)
      })
      .catch((err) => {
        if (!active) return
        const msg = err instanceof Error ? err.message : String(err)
        const isChunkLoadError = msg.includes('Failed to fetch dynamically imported module') ||
          msg.includes('Loading chunk') ||
          msg.includes('Loading CSS chunk')
        setError(
          isChunkLoadError
            ? 'Ошибка загрузки диаграммы. Обновите страницу (Ctrl+Shift+R).'
            : msg,
        )
      })

    return () => {
      active = false
    }
  }, [id, code, resolvedTheme])

  if (error) {
    return (
      <pre
        className="overflow-x-auto rounded p-3 bg-muted text-muted-foreground text-sm"
        data-mermaid-error
      >
        <code>{code}</code>
        <div className="mt-2 text-destructive text-xs">{error}</div>
      </pre>
    )
  }

  return (
    <div
      ref={containerRef}
      className="my-4 flex justify-center [&_svg]:max-w-full [&_svg]:h-auto"
      data-mermaid-diagram
    />
  )
}
