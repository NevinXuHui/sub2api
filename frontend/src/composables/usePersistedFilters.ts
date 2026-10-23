import { isRef, onBeforeMount, watch } from 'vue'

const PREFIX = 'sub2api:page-filters:v1:'
const blockedKeys = new Set(['__proto__', 'constructor', 'prototype'])

// Only explicitly registered filter state is stored, never forms, rows or credentials.
function clean(value: unknown, depth = 0): unknown {
  if (depth > 8) return undefined
  if (value === null || typeof value === 'boolean') return value
  if (typeof value === 'string') return value.slice(0, 10_000)
  if (typeof value === 'number') return Number.isFinite(value) ? value : undefined
  if (value instanceof Set) return [...value].map(item => clean(item, depth + 1))
  if (Array.isArray(value)) return value.slice(0, 1000).map(item => clean(item, depth + 1))
  if (value && typeof value === 'object') {
    return Object.fromEntries(Object.entries(value)
      .filter(([key]) => !blockedKeys.has(key))
      .map(([key, item]) => [key, clean(item, depth + 1)]))
  }
  return undefined
}

export function pageFilterStorageKey(page: string): string {
  // auth_user is populated before the authenticated route is mounted.
  const user = JSON.parse(localStorage.getItem('auth_user') || 'null')
  const scope = typeof user?.id === 'number' || typeof user?.id === 'string' ? String(user.id) : 'guest'
  return `${PREFIX}${scope}:${page}`
}

function compatible(value: unknown, current: unknown): boolean {
  if (current === null || current === undefined) return true
  if (current instanceof Set || Array.isArray(current)) return Array.isArray(value)
  if (typeof current === 'object') return value !== null && typeof value === 'object' && !Array.isArray(value)
  // Select controls can represent an ID as either a number or a string.
  if (typeof current === 'string' || typeof current === 'number') return typeof value === 'string' || typeof value === 'number'
  return typeof value === typeof current
}

function restoreObject(current: Record<string, unknown>, saved: Record<string, unknown>): Record<string, unknown> {
  const restored = { ...current }
  for (const [key, value] of Object.entries(saved)) {
    if (blockedKeys.has(key) || !compatible(value, current[key])) continue
    restored[key] = value
  }
  return restored
}

export function usePersistedFilters(
  page: string,
  bindings: Record<string, unknown>,
  options: { skipRestore?: () => boolean } = {}
): void {
  onBeforeMount(() => {
    let key: string
    try { key = pageFilterStorageKey(page) } catch { return }
    try {
      const raw = localStorage.getItem(key)
      const saved = raw ? clean(JSON.parse(raw)) : null
      if (!options.skipRestore?.() && saved && typeof saved === 'object' && !Array.isArray(saved)) {
        for (const [name, binding] of Object.entries(bindings)) {
          if (!Object.prototype.hasOwnProperty.call(saved, name)) continue
          const value = (saved as Record<string, unknown>)[name]
          const current = isRef(binding) ? binding.value : binding
          if (!compatible(value, current)) continue
          if (current instanceof Set && Array.isArray(value)) {
            current.clear()
            value.forEach(item => current.add(item))
          } else if (isRef(binding)) {
            binding.value = current && typeof current === 'object' && !Array.isArray(current)
              ? restoreObject(current as Record<string, unknown>, value as Record<string, unknown>) : value
          } else if (current && typeof current === 'object' && value && typeof value === 'object') {
            Object.assign(current, restoreObject(current as Record<string, unknown>, value as Record<string, unknown>))
          }
        }
      }
    } catch {
      // Corrupt or unavailable browser storage must not prevent page loading.
    }
    watch(() => Object.fromEntries(Object.entries(bindings).map(([name, binding]) =>
      [name, isRef(binding) ? binding.value : binding])), value => {
      try { localStorage.setItem(key, JSON.stringify(clean(value))) } catch { /* storage quota/private mode */ }
    }, { deep: true, flush: 'sync' })
  })
}
