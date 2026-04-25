import { Toast as T } from 'radix-ui'
import { X } from 'lucide-react'
import { dismissToast, useToastsSnapshot, type ToastRecord } from '@/hooks/use-toast'
import { cn } from '@/lib/utils'

function ToastItem({ item }: { item: ToastRecord }) {
  return (
    <T.Root
      type="foreground"
      duration={item.variant === 'destructive' ? 8000 : 5000}
      defaultOpen
      onOpenChange={(open) => {
        if (!open) dismissToast(item.id)
      }}
      className={cn(
        'group pointer-events-auto relative flex w-[calc(100%-2rem)] max-w-md items-start gap-3 rounded-lg border p-4 pr-10 shadow-lg transition-all sm:w-full',
        item.variant === 'destructive'
          ? 'border-destructive bg-destructive text-destructive-foreground'
          : 'border-border bg-popover text-popover-foreground',
      )}
    >
      <div className="grid min-w-0 flex-1 gap-1">
        {item.title ? (
          <T.Title className="text-sm font-semibold">{item.title}</T.Title>
        ) : null}
        <T.Description className="break-words text-sm opacity-90">{item.description}</T.Description>
      </div>
      <T.Close
        className="absolute right-2 top-2 rounded-md p-1 opacity-70 transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring"
        aria-label="Закрыть"
      >
        <X className="size-4 shrink-0" />
      </T.Close>
    </T.Root>
  )
}

/** Viewport + список тостов; оборачивайте приложение в `<Toast.Provider>` из `radix-ui`. */
export function ToasterViewport() {
  const list = useToastsSnapshot()
  return (
    <>
      {list.map((item) => (
        <ToastItem key={item.id} item={item} />
      ))}
      <T.Viewport className="fixed bottom-0 right-0 z-[100] flex max-h-[min(50dvh,320px)] w-full list-none flex-col gap-2 overflow-x-hidden overflow-y-auto p-4 sm:bottom-4 sm:right-4 sm:max-w-[min(420px,calc(100vw-2rem))]" />
    </>
  )
}
