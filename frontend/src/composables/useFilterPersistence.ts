/**
 * useFilterPersistence - Universal filter persistence composable
 *
 * Provides automatic localStorage persistence for page filters and settings.
 * Each page gets a unique storage key based on its route path.
 *
 * @example
 * const { filters, updateFilter, resetFilters } = useFilterPersistence('admin-users', {
 *   search: '',
 *   status: 'all',
 *   page: 1
 * })
 */

import { ref, watch, type Ref } from 'vue'

const STORAGE_PREFIX = 'filter_'
const STORAGE_VERSION = 'v1'

export interface FilterPersistenceOptions<T> {
  /** Storage key (e.g., 'admin-users', 'admin-usage') */
  key: string
  /** Default filter values */
  defaults: T
  /** Optional: fields to exclude from persistence */
  excludeFields?: (keyof T)[]
  /** Optional: TTL in milliseconds (default: 7 days) */
  ttl?: number
}

interface StoredData<T> {
  version: string
  timestamp: number
  data: T
}

export function useFilterPersistence<T extends Record<string, any>>(
  key: string,
  defaults: T,
  options?: {
    excludeFields?: (keyof T)[]
    ttl?: number
  }
) {
  const storageKey = `${STORAGE_PREFIX}${STORAGE_VERSION}_${key}`
  const ttl = options?.ttl ?? 7 * 24 * 60 * 60 * 1000 // 7 days default
  const excludeFields = new Set(options?.excludeFields ?? [])

  // Load saved filters from localStorage
  const loadFilters = (): T => {
    try {
      const saved = localStorage.getItem(storageKey)
      if (!saved) return { ...defaults }

      const stored: StoredData<Partial<T>> = JSON.parse(saved)

      // Version check
      if (stored.version !== STORAGE_VERSION) {
        localStorage.removeItem(storageKey)
        return { ...defaults }
      }

      // TTL check
      if (ttl > 0 && Date.now() - stored.timestamp > ttl) {
        localStorage.removeItem(storageKey)
        return { ...defaults }
      }

      // Merge saved data with defaults (defaults take precedence for missing keys)
      return { ...defaults, ...stored.data }
    } catch (error) {
      console.warn(`Failed to load filters for ${key}:`, error)
      return { ...defaults }
    }
  }

  // Save filters to localStorage
  const saveFilters = (data: T) => {
    try {
      // Filter out excluded fields
      const dataToSave: Partial<T> = {}
      for (const [k, v] of Object.entries(data)) {
        if (!excludeFields.has(k as keyof T)) {
          dataToSave[k as keyof T] = v
        }
      }

      const stored: StoredData<Partial<T>> = {
        version: STORAGE_VERSION,
        timestamp: Date.now(),
        data: dataToSave
      }

      localStorage.setItem(storageKey, JSON.stringify(stored))
    } catch (error) {
      console.warn(`Failed to save filters for ${key}:`, error)
    }
  }

  // Initialize reactive filters
  const filters = ref<T>(loadFilters()) as Ref<T>

  // Watch for changes and auto-save
  watch(
    filters,
    (newFilters) => {
      saveFilters(newFilters)
    },
    { deep: true }
  )

  // Update a single filter field
  const updateFilter = <K extends keyof T>(field: K, value: T[K]) => {
    filters.value[field] = value
  }

  // Batch update multiple fields
  const updateFilters = (updates: Partial<T>) => {
    Object.assign(filters.value, updates)
  }

  // Reset to defaults
  const resetFilters = () => {
    filters.value = { ...defaults }
    localStorage.removeItem(storageKey)
  }

  // Clear only this page's storage
  const clearStorage = () => {
    localStorage.removeItem(storageKey)
  }

  return {
    filters,
    updateFilter,
    updateFilters,
    resetFilters,
    clearStorage
  }
}

/**
 * Clear all filter persistence data across the app
 */
export function clearAllFilterPersistence() {
  const keys = Object.keys(localStorage)
  const filterKeys = keys.filter(k => k.startsWith(STORAGE_PREFIX))
  filterKeys.forEach(k => localStorage.removeItem(k))
}
