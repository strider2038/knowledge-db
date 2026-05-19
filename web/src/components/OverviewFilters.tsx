import { useMemo, useState, type ReactNode } from 'react'
import { Filter, Plus, Search, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { cn } from '@/lib/utils'
import { getTypeButtonClass } from '@/lib/type-styles'
import { getLabelChipClass } from '@/lib/label-styles'

const NODE_TYPES = ['article', 'link', 'note'] as const

const TYPE_LABELS: Record<(typeof NODE_TYPES)[number], string> = {
  article: 'статья',
  link: 'ссылка',
  note: 'заметка',
}

const selectClassName =
  'h-9 w-full min-w-0 rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2'

const searchClassName =
  'h-9 w-full min-w-0 rounded-md border border-input bg-background px-3 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2'

const dropdownSearchClassName =
  'h-9 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2'

export interface OverviewFiltersProps {
  q: string
  onSearchChange: (value: string) => void
  typeFilter: string[]
  onToggleType: (type: string) => void
  manualProcessedFilter: '' | 'true' | 'false'
  onManualProcessedChange: (value: '' | 'true' | 'false') => void
  labelFilter: string[]
  labelSuggestions: string[]
  onAddLabelFilter: (label: string) => void
  onRemoveLabelFilter: (label: string) => void
  onToggleLabelSuggestion: (label: string) => void
  onClearAllFilters: () => void
}

function FilterSection({
  title,
  description,
  children,
}: {
  title: string
  description?: string
  children: ReactNode
}) {
  return (
    <section className="space-y-2">
      <div className="space-y-0.5">
        <h3 className="text-sm font-medium leading-none">{title}</h3>
        {description ? (
          <p className="text-xs text-muted-foreground">{description}</p>
        ) : null}
      </div>
      {children}
    </section>
  )
}

function LabelFilterPicker({
  labelFilter,
  labelSuggestions,
  onAddLabelFilter,
  onRemoveLabelFilter,
  onToggleLabelSuggestion,
}: Pick<
  OverviewFiltersProps,
  'labelFilter' | 'labelSuggestions' | 'onAddLabelFilter' | 'onRemoveLabelFilter' | 'onToggleLabelSuggestion'
>) {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')

  const filteredSuggestions = useMemo(() => {
    const q = query.trim().toLocaleLowerCase()
    return labelSuggestions.filter((label) =>
      q ? label.toLocaleLowerCase().includes(q) : true
    )
  }, [labelSuggestions, query])

  const trimmedQuery = query.trim()
  const canAddCustom =
    trimmedQuery.length > 0 &&
    !trimmedQuery.includes(',') &&
    !labelFilter.some((l) => l.toLocaleLowerCase() === trimmedQuery.toLocaleLowerCase()) &&
    !labelSuggestions.some((l) => l.toLocaleLowerCase() === trimmedQuery.toLocaleLowerCase())

  const handleOpenChange = (next: boolean) => {
    setOpen(next)
    if (!next) setQuery('')
  }

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {labelFilter.map((label) => (
        <button
          key={label}
          type="button"
          onClick={() => onRemoveLabelFilter(label)}
          className={cn(getLabelChipClass(label), 'gap-1 pr-1.5')}
          title="Снять фильтр"
        >
          {label}
          <X className="size-3 opacity-70" aria-hidden />
        </button>
      ))}
      <DropdownMenu open={open} onOpenChange={handleOpenChange}>
        <DropdownMenuTrigger asChild>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-7 gap-1 border-dashed px-2 text-muted-foreground"
            aria-label="Добавить метку в фильтр"
          >
            <Plus className="size-3.5" />
            <span className="text-xs">Метка</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-64 p-0" sideOffset={6}>
          <div className="border-b p-2">
            <input
              type="search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && canAddCustom) {
                  e.preventDefault()
                  onAddLabelFilter(trimmedQuery)
                  setQuery('')
                }
              }}
              placeholder="Найти или ввести…"
              className={dropdownSearchClassName}
              aria-label="Поиск метки"
              autoFocus
            />
          </div>
          <div className="max-h-56 overflow-y-auto p-1">
            {filteredSuggestions.length > 0 ? (
              filteredSuggestions.map((label) => {
                const checked = labelFilter.some(
                  (l) => l.toLocaleLowerCase() === label.toLocaleLowerCase()
                )
                return (
                  <DropdownMenuCheckboxItem
                    key={label}
                    checked={checked}
                    onCheckedChange={() => onToggleLabelSuggestion(label)}
                    className="py-2 focus:bg-muted/60"
                  >
                    <span className={getLabelChipClass(label)}>{label}</span>
                  </DropdownMenuCheckboxItem>
                )
              })
            ) : (
              <p className="px-2 py-3 text-center text-xs text-muted-foreground">
                {labelSuggestions.length === 0
                  ? 'В базе пока нет меток'
                  : 'Ничего не найдено'}
              </p>
            )}
          </div>
          {canAddCustom ? (
            <>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                className="gap-2 py-2 focus:bg-muted/60"
                onSelect={() => {
                  onAddLabelFilter(trimmedQuery)
                  setQuery('')
                }}
              >
                <span className="text-sm">Добавить</span>
                <span className={getLabelChipClass(trimmedQuery)}>{trimmedQuery}</span>
              </DropdownMenuItem>
            </>
          ) : null}
          {labelFilter.length > 0 ? (
            <>
              <DropdownMenuSeparator />
              <DropdownMenuLabel className="text-xs font-normal text-muted-foreground">
                Выбрано: {labelFilter.length} (все должны быть на узле)
              </DropdownMenuLabel>
            </>
          ) : null}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}

function OverviewFiltersPanel({
  typeFilter,
  onToggleType,
  manualProcessedFilter,
  onManualProcessedChange,
  labelFilter,
  labelSuggestions,
  onAddLabelFilter,
  onRemoveLabelFilter,
  onToggleLabelSuggestion,
}: Pick<
  OverviewFiltersProps,
  | 'typeFilter'
  | 'onToggleType'
  | 'manualProcessedFilter'
  | 'onManualProcessedChange'
  | 'labelFilter'
  | 'labelSuggestions'
  | 'onAddLabelFilter'
  | 'onRemoveLabelFilter'
  | 'onToggleLabelSuggestion'
>) {
  return (
    <div className="flex flex-col gap-5">
      <FilterSection title="Тип контента">
        <div className="flex flex-wrap gap-1.5">
          {NODE_TYPES.map((t) => {
            const isActive = typeFilter.includes(t)
            return (
              <Button
                key={t}
                type="button"
                variant="outline"
                size="sm"
                className={getTypeButtonClass(t, isActive)}
                onClick={() => onToggleType(t)}
              >
                {TYPE_LABELS[t]}
              </Button>
            )
          })}
        </div>
      </FilterSection>

      <FilterSection title="Ручная проверка">
        <select
          value={manualProcessedFilter}
          onChange={(e) =>
            onManualProcessedChange(e.target.value as '' | 'true' | 'false')
          }
          className={selectClassName}
          aria-label="Фильтр по ручной проверке"
        >
          <option value="">Все записи</option>
          <option value="true">Проверено вручную</option>
          <option value="false">Не проверено</option>
        </select>
      </FilterSection>

      <FilterSection
        title="Метки"
        description="Показываются узлы, у которых есть все выбранные метки (AND)."
      >
        <LabelFilterPicker
          labelFilter={labelFilter}
          labelSuggestions={labelSuggestions}
          onAddLabelFilter={onAddLabelFilter}
          onRemoveLabelFilter={onRemoveLabelFilter}
          onToggleLabelSuggestion={onToggleLabelSuggestion}
        />
      </FilterSection>
    </div>
  )
}

function ActiveFiltersSummary({
  typeFilter,
  manualProcessedFilter,
  onToggleType,
  onManualProcessedChange,
  onClearAllFilters,
}: Pick<
  OverviewFiltersProps,
  | 'typeFilter'
  | 'manualProcessedFilter'
  | 'onToggleType'
  | 'onManualProcessedChange'
  | 'onClearAllFilters'
>) {
  return (
    <div className="flex flex-wrap items-center gap-2 rounded-md border border-dashed border-border/80 bg-muted/20 px-2 py-1.5">
      <span className="text-xs font-medium text-muted-foreground">Активно:</span>
      {typeFilter.map((t) => (
        <button
          key={t}
          type="button"
          onClick={() => onToggleType(t)}
          className={cn(
            'inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs',
            getTypeButtonClass(t as (typeof NODE_TYPES)[number], true)
          )}
        >
          {TYPE_LABELS[t as (typeof NODE_TYPES)[number]] ?? t}
          <X className="size-3" aria-hidden />
        </button>
      ))}
      {manualProcessedFilter === 'true' ? (
        <button
          type="button"
          onClick={() => onManualProcessedChange('')}
          className="inline-flex items-center gap-1 rounded-full border border-border bg-background px-2 py-0.5 text-xs"
        >
          Проверено
          <X className="size-3" aria-hidden />
        </button>
      ) : null}
      {manualProcessedFilter === 'false' ? (
        <button
          type="button"
          onClick={() => onManualProcessedChange('')}
          className="inline-flex items-center gap-1 rounded-full border border-border bg-background px-2 py-0.5 text-xs"
        >
          Не проверено
          <X className="size-3" aria-hidden />
        </button>
      ) : null}
      <Button
        type="button"
        variant="ghost"
        size="sm"
        className="h-7 px-2 text-xs text-muted-foreground"
        onClick={onClearAllFilters}
      >
        Сбросить всё
      </Button>
    </div>
  )
}

export function OverviewFilters({
  q,
  onSearchChange,
  typeFilter,
  onToggleType,
  manualProcessedFilter,
  onManualProcessedChange,
  labelFilter,
  labelSuggestions,
  onAddLabelFilter,
  onRemoveLabelFilter,
  onToggleLabelSuggestion,
  onClearAllFilters,
}: OverviewFiltersProps) {
  const [filtersSheetOpen, setFiltersSheetOpen] = useState(false)

  const activeFilterCount =
    typeFilter.length +
    (manualProcessedFilter !== '' ? 1 : 0) +
    labelFilter.length

  const hasSecondaryFilters = activeFilterCount > 0
  const hasNonLabelFilters =
    typeFilter.length > 0 || manualProcessedFilter !== ''

  const panelProps = {
    typeFilter,
    onToggleType,
    manualProcessedFilter,
    onManualProcessedChange,
    labelFilter,
    labelSuggestions,
    onAddLabelFilter,
    onRemoveLabelFilter,
    onToggleLabelSuggestion,
  }

  return (
    <div className="mb-4 space-y-3">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
        <div className="relative min-w-0 flex-1">
          <Search
            className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground"
            aria-hidden
          />
          <input
            type="search"
            placeholder="Поиск по названию, ключевым словам…"
            value={q}
            onChange={(e) => onSearchChange(e.target.value)}
            className={cn(searchClassName, 'pl-9')}
            aria-label="Поиск узлов"
          />
        </div>
        <Sheet open={filtersSheetOpen} onOpenChange={setFiltersSheetOpen}>
          <SheetTrigger asChild>
            <Button
              type="button"
              variant="outline"
              className="shrink-0 md:hidden"
              aria-label={
                activeFilterCount > 0
                  ? `Фильтры, активно: ${activeFilterCount}`
                  : 'Фильтры'
              }
            >
              <Filter className="size-4" />
              Фильтры
              {activeFilterCount > 0 ? (
                <span className="ml-1 inline-flex min-w-5 items-center justify-center rounded-full bg-primary px-1.5 py-0.5 text-[10px] font-medium text-primary-foreground">
                  {activeFilterCount}
                </span>
              ) : null}
            </Button>
          </SheetTrigger>
          <SheetContent side="right" className="flex w-full flex-col p-0 sm:max-w-md">
            <SheetHeader className="border-b p-4 pr-14 text-left">
              <SheetTitle>Фильтры</SheetTitle>
            </SheetHeader>
            <div className="flex-1 overflow-y-auto p-4">
              <OverviewFiltersPanel {...panelProps} />
            </div>
            {hasSecondaryFilters ? (
              <div className="border-t p-4">
                <Button
                  type="button"
                  variant="outline"
                  className="w-full"
                  onClick={() => {
                    onClearAllFilters()
                    setFiltersSheetOpen(false)
                  }}
                >
                  Сбросить все фильтры
                </Button>
              </div>
            ) : null}
          </SheetContent>
        </Sheet>
      </div>

      {hasNonLabelFilters ? (
        <ActiveFiltersSummary
          typeFilter={typeFilter}
          manualProcessedFilter={manualProcessedFilter}
          onToggleType={onToggleType}
          onManualProcessedChange={onManualProcessedChange}
          onClearAllFilters={onClearAllFilters}
        />
      ) : null}

      {labelFilter.length > 0 && !filtersSheetOpen ? (
        <div className="md:hidden">
          <LabelFilterPicker
            labelFilter={labelFilter}
            labelSuggestions={labelSuggestions}
            onAddLabelFilter={onAddLabelFilter}
            onRemoveLabelFilter={onRemoveLabelFilter}
            onToggleLabelSuggestion={onToggleLabelSuggestion}
          />
        </div>
      ) : null}

      <div className="hidden rounded-lg border border-border/60 bg-muted/20 p-3 md:block">
        <OverviewFiltersPanel {...panelProps} />
        {hasSecondaryFilters ? (
          <div className="mt-3 flex justify-end border-t border-border/60 pt-3">
            <Button type="button" variant="ghost" size="sm" onClick={onClearAllFilters}>
              Сбросить все фильтры
            </Button>
          </div>
        ) : null}
      </div>
    </div>
  )
}
