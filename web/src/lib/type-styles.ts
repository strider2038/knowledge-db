/**
 * Цвета типов узлов (article, link, note) для фильтров и селекторов.
 * Используются на странице Обзор и в форме Добавить.
 * Правила: .cursor/rules/web-type-colors.mdc
 */

export const TYPE_BUTTON_CLASSES: Record<
  'article' | 'link' | 'note' | 'auto',
  { active: string; inactive: string }
> = {
  article: {
    active:
      'bg-blue-500/20 text-blue-700 border-blue-500/40 hover:bg-blue-500/30 dark:text-blue-300 dark:border-blue-400/50',
    inactive:
      'border-blue-500/30 text-blue-600 hover:bg-blue-500/10 dark:text-blue-400',
  },
  link: {
    active:
      'bg-green-500/20 text-green-700 border-green-500/40 hover:bg-green-500/30 dark:text-green-300 dark:border-green-400/50',
    inactive:
      'border-green-500/30 text-green-600 hover:bg-green-500/10 dark:text-green-400',
  },
  note: {
    active:
      'bg-amber-500/20 text-amber-700 border-amber-500/40 hover:bg-amber-500/30 dark:text-amber-300 dark:border-amber-400/50',
    inactive:
      'border-amber-500/30 text-amber-600 hover:bg-amber-500/10 dark:text-amber-400',
  },
  auto: {
    active: 'bg-muted text-muted-foreground border-border',
    inactive: 'border-border text-muted-foreground hover:bg-muted',
  },
}

export function getTypeButtonClass(
  type: 'article' | 'link' | 'note' | 'auto',
  isActive: boolean
): string {
  const { active, inactive } = TYPE_BUTTON_CLASSES[type]
  return isActive ? active : inactive
}

/** Цвета для badge (чип в таблице узлов). */
export const TYPE_BADGE_COLORS: Record<string, string> = {
  article: 'bg-blue-500/20 text-blue-700 dark:text-blue-300',
  link: 'bg-green-500/20 text-green-700 dark:text-green-300',
  note: 'bg-amber-500/20 text-amber-700 dark:text-amber-300',
}

export function getTypeBadgeColor(type: string): string {
  return TYPE_BADGE_COLORS[type] ?? 'bg-muted text-muted-foreground'
}
