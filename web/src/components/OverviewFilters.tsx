import { useMemo, useState } from 'react'
import { ChevronDown, Filter, Plus, Search, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
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

function ToolbarDivider({ className }: { className?: string }) {
  return (
    <span
      className={cn('hidden text-muted-foreground/40 sm:inline', className)}
      aria-hidden
    >
      |
    </span>
  )
}

function ManualProcessedFilterDropdown({
  value,
  onChange,
}: {
  value: '' | 'true' | 'false'
  onChange: (value: '' | 'true' | 'false') => void
}) {
  const triggerLabel =
    value === 'true'
      ? 'Проверено'
      : value === 'false'
        ? 'Не проверено'
        : 'Проверка'

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className={cn('h-8 gap-1', value !== '' && 'border-primary/50 bg-primary/5')}
          aria-label="Фильтр по ручной проверке"
        >
          {triggerLabel}
          <ChevronDown className="size-3.5 opacity-60" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="min-w-44">
        <DropdownMenuRadioGroup
          value={value === '' ? 'all' : value}
          onValueChange={(next) =>
            onChange(next === 'all' ? '' : (next as 'true' | 'false'))
          }
        >
          <DropdownMenuRadioItem value="all">Все записи</DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="true">Проверено вручную</DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="false">Не проверено</DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function LabelFilterPicker({
  labelFilter,
  labelSuggestions,
  onAddLabelFilter,
  onRemoveLabelFilter,
  onToggleLabelSuggestion,
  showChips = true,
}: Pick<
  OverviewFiltersProps,
  | 'labelFilter'
  | 'labelSuggestions'
  | 'onAddLabelFilter'
  | 'onRemoveLabelFilter'
  | 'onToggleLabelSuggestion'
> & {
  showChips?: boolean
}) {
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
      {showChips
        ? labelFilter.map((label) => (
            <button
              key={label}
              type="button"
              onClick={() => onRemoveLabelFilter(label)}
              className={cn(getLabelChipClass(label), 'gap-1 pr-1.5')}
              title="Снять фильтр по метке"
            >
              {label}
              <X className="size-3 opacity-70" aria-hidden />
            </button>
          ))
        : null}
      <DropdownMenu open={open} onOpenChange={handleOpenChange}>
        <DropdownMenuTrigger asChild>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-7 gap-1 border-dashed px-2 text-muted-foreground"
            aria-label="Добавить метку в фильтр. Несколько меток объединяются по AND."
            title="Фильтр по меткам (AND): на узле должны быть все выбранные"
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
                Выбрано: {labelFilter.length} (AND)
              </DropdownMenuLabel>
            </>
          ) : null}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}

function FiltersToolbar({
  typeFilter,
  onToggleType,
  manualProcessedFilter,
  onManualProcessedChange,
  labelFilter,
  labelSuggestions,
  onAddLabelFilter,
  onRemoveLabelFilter,
  onToggleLabelSuggestion,
  showLabelChips = true,
  showClear = false,
  onClearAllFilters,
  className,
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
  | 'onClearAllFilters'
> & {
  showLabelChips?: boolean
  showClear?: boolean
  className?: string
}) {
  return (
    <div className={cn('flex flex-wrap items-center gap-2', className)}>
      <div className="flex flex-wrap items-center gap-1.5">
        {NODE_TYPES.map((t) => {
          const isActive = typeFilter.includes(t)
          return (
            <Button
              key={t}
              type="button"
              variant="outline"
              size="sm"
              className={cn('h-8', getTypeButtonClass(t, isActive))}
              onClick={() => onToggleType(t)}
            >
              {TYPE_LABELS[t]}
            </Button>
          )
        })}
      </div>
      <ToolbarDivider />
      <ManualProcessedFilterDropdown
        value={manualProcessedFilter}
        onChange={onManualProcessedChange}
      />
      <ToolbarDivider />
      <LabelFilterPicker
        labelFilter={labelFilter}
        labelSuggestions={labelSuggestions}
        onAddLabelFilter={onAddLabelFilter}
        onRemoveLabelFilter={onRemoveLabelFilter}
        onToggleLabelSuggestion={onToggleLabelSuggestion}
        showChips={showLabelChips}
      />
      {showClear ? (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-8 px-2 text-xs text-muted-foreground"
          onClick={onClearAllFilters}
        >
          Сбросить
        </Button>
      ) : null}
    </div>
  )
}

function ActiveFiltersSummary({
  typeFilter,
  manualProcessedFilter,
  labelFilter,
  onToggleType,
  onManualProcessedChange,
  onRemoveLabelFilter,
  onClearAllFilters,
}: Pick<
  OverviewFiltersProps,
  | 'typeFilter'
  | 'manualProcessedFilter'
  | 'labelFilter'
  | 'onToggleType'
  | 'onManualProcessedChange'
  | 'onRemoveLabelFilter'
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
      {labelFilter.map((label) => (
        <button
          key={label}
          type="button"
          onClick={() => onRemoveLabelFilter(label)}
          className={cn(getLabelChipClass(label), 'gap-1 pr-1.5')}
        >
          {label}
          <X className="size-3 opacity-70" aria-hidden />
        </button>
      ))}
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

  const toolbarProps = {
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
  }

  return (
    <div className="mb-4 space-y-2">
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
              <FiltersToolbar
                {...toolbarProps}
                showLabelChips
                className="flex-col items-stretch gap-3"
              />
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

      <div className="hidden md:block">
        <FiltersToolbar {...toolbarProps} showLabelChips showClear={hasSecondaryFilters} />
      </div>

      {hasSecondaryFilters && !filtersSheetOpen ? (
        <div className="md:hidden">
          <ActiveFiltersSummary {...toolbarProps} />
        </div>
      ) : null}
    </div>
  )
}
