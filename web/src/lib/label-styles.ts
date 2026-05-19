/**
 * Цвета личных меток узлов (labels) — отдельно от типов и keywords.
 */

const LABEL_CHIP_CLASSES = [
  'bg-violet-500/20 text-violet-800 border-violet-500/35 dark:text-violet-200 dark:border-violet-400/45',
  'bg-rose-500/20 text-rose-800 border-rose-500/35 dark:text-rose-200 dark:border-rose-400/45',
  'bg-cyan-500/20 text-cyan-800 border-cyan-500/35 dark:text-cyan-200 dark:border-cyan-400/45',
  'bg-orange-500/20 text-orange-800 border-orange-500/35 dark:text-orange-200 dark:border-orange-400/45',
  'bg-teal-500/20 text-teal-800 border-teal-500/35 dark:text-teal-200 dark:border-teal-400/45',
  'bg-fuchsia-500/20 text-fuchsia-800 border-fuchsia-500/35 dark:text-fuchsia-200 dark:border-fuchsia-400/45',
  'bg-lime-500/20 text-lime-800 border-lime-500/35 dark:text-lime-200 dark:border-lime-400/45',
  'bg-sky-500/20 text-sky-800 border-sky-500/35 dark:text-sky-200 dark:border-sky-400/45',
  'bg-pink-500/20 text-pink-800 border-pink-500/35 dark:text-pink-200 dark:border-pink-400/45',
  'bg-indigo-500/20 text-indigo-800 border-indigo-500/35 dark:text-indigo-200 dark:border-indigo-400/45',
] as const

function hashLabel(label: string): number {
  let hash = 2166136261
  for (let i = 0; i < label.length; i++) {
    hash ^= label.charCodeAt(i)
    hash = Math.imul(hash, 16777619)
  }

  return hash >>> 0
}

export function getLabelChipClass(label: string): string {
  const base =
    'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium'
  const idx = hashLabel(label) % LABEL_CHIP_CLASSES.length

  return `${base} ${LABEL_CHIP_CLASSES[idx]}`
}
