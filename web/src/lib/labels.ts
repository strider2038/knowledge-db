export function normalizeLabel(value: string): string {
  return value.trim()
}

export function dedupeLabels(values: string[]): string[] {
  const seen = new Set<string>()
  const result: string[] = []
  for (const value of values) {
    const normalized = normalizeLabel(value)
    if (!normalized || normalized.includes(',')) continue
    const key = normalized.toLocaleLowerCase()
    if (seen.has(key)) continue
    seen.add(key)
    result.push(normalized)
  }
  return result
}
