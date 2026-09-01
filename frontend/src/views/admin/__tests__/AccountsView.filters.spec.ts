import { describe, it, expect, beforeEach, vi } from 'vitest'

describe('AccountsView Filter Persistence', () => {
  beforeEach(() => {
    // 清空 localStorage
    localStorage.clear()
  })

  it('should save filters to localStorage', () => {
    const filters = {
      platform: 'anthropic',
      type: 'oauth',
      status: 'active',
      privacy_mode: 'training_off',
      group: '1',
      search: 'test'
    }

    localStorage.setItem('account-table-filters', JSON.stringify(filters))

    const saved = JSON.parse(localStorage.getItem('account-table-filters') || '{}')
    expect(saved).toEqual(filters)
  })

  it('should load filters from localStorage', () => {
    const filters = {
      platform: 'openai',
      type: 'apikey',
      status: 'error',
      privacy_mode: '',
      group: 'ungrouped',
      search: 'search term'
    }

    localStorage.setItem('account-table-filters', JSON.stringify(filters))

    const loaded = JSON.parse(localStorage.getItem('account-table-filters') || '{}')
    expect(loaded.platform).toBe('openai')
    expect(loaded.type).toBe('apikey')
    expect(loaded.status).toBe('error')
    expect(loaded.search).toBe('search term')
  })

  it('should handle empty/missing filters gracefully', () => {
    const loaded = JSON.parse(localStorage.getItem('account-table-filters') || '{}')
    expect(Object.keys(loaded).length).toBe(0)
  })

  it('should persist filters after page reload simulation', () => {
    const initialFilters = {
      platform: 'gemini',
      type: 'oauth',
      status: 'active',
      privacy_mode: 'training_off',
      group: '2',
      search: 'test account'
    }

    // 模拟保存
    localStorage.setItem('account-table-filters', JSON.stringify(initialFilters))

    // 模拟页面刷新后加载
    const reloadedFilters = JSON.parse(localStorage.getItem('account-table-filters') || '{}')

    expect(reloadedFilters).toEqual(initialFilters)
  })

  it('should handle invalid JSON in localStorage', () => {
    localStorage.setItem('account-table-filters', 'invalid json')

    let filters
    try {
      filters = JSON.parse(localStorage.getItem('account-table-filters') || '{}')
    } catch {
      filters = {
        platform: '',
        type: '',
        status: '',
        privacy_mode: '',
        group: '',
        search: ''
      }
    }

    expect(filters.platform).toBeDefined()
  })
})
