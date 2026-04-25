import * as React from 'react'

export type ToastPayload = {
  title?: string
  description: string
  variant?: 'default' | 'destructive'
}

export type ToastRecord = ToastPayload & { id: string }

const TOAST_LIMIT = 5

let memoryToasts: ToastRecord[] = []
const listeners = new Set<() => void>()

function emit() {
  listeners.forEach((l) => l())
}

/** Показать всплывающее уведомление (Radix Toast, низ экрана). */
export function toast(payload: ToastPayload) {
  const id = `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
  memoryToasts = [...memoryToasts, { id, ...payload }].slice(-TOAST_LIMIT)
  emit()
}

export function dismissToast(id: string) {
  memoryToasts = memoryToasts.filter((t) => t.id !== id)
  emit()
}

export function subscribeToasts(listener: () => void) {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

export function getToastsSnapshot(): ToastRecord[] {
  return [...memoryToasts]
}

/** Для тестов: сброс очереди уведомлений. */
export function clearAllToasts() {
  memoryToasts = []
  emit()
}

export function useToastsSnapshot() {
  const [list, setList] = React.useState<ToastRecord[]>(() => getToastsSnapshot())
  React.useEffect(() => {
    const sync = () => setList(getToastsSnapshot())
    return subscribeToasts(sync)
  }, [])
  return list
}
