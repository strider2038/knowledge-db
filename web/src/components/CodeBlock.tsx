import * as React from 'react'
import { Copy, Check } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

interface CodeBlockProps {
  children: React.ReactNode
  preProps?: React.ComponentProps<'pre'>
  className?: string
}

function getCodeText(children: React.ReactNode): string {
  const child = React.Children.only(children) as
    | React.ReactElement<{ children?: React.ReactNode }>
    | undefined
  if (!child?.props?.children) return ''
  const code = child.props.children
  return typeof code === 'string' ? code : Array.isArray(code) ? code.join('') : ''
}

export function CodeBlock({ children, preProps, className }: CodeBlockProps) {
  const [copied, setCopied] = React.useState(false)
  const codeText = getCodeText(children)

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(codeText)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // ignore
    }
  }

  return (
    <div
      data-code-block
      className={cn(
        'relative overflow-hidden rounded-lg border shadow-sm p-4 pr-12',
        className
      )}
    >
      <div className="absolute right-2 top-2 z-10">
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={handleCopy}
              className="h-7 w-7 text-muted-foreground hover:text-foreground"
              aria-label={copied ? 'Скопировано' : 'Копировать'}
            >
              {copied ? (
                <Check className="size-3.5 text-green-600 dark:text-green-400" />
              ) : (
                <Copy className="size-3.5" />
              )}
            </Button>
          </TooltipTrigger>
          <TooltipContent side="left">
            {copied ? 'Скопировано' : 'Копировать'}
          </TooltipContent>
        </Tooltip>
      </div>
      <pre
        {...preProps}
        className={cn(
          'overflow-x-auto [&>code]:block',
          preProps?.className
        )}
      >
        {children}
      </pre>
    </div>
  )
}
