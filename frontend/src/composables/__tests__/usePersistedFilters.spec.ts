import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, onMounted, reactive, ref } from 'vue'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import { pageFilterStorageKey, usePersistedFilters } from '../usePersistedFilters'

enableAutoUnmount(afterEach)

function harness(page = 'admin/accounts', skipRestore = false) {
  let state!: { search: ReturnType<typeof ref<string>>; filters: { status: string; group: number | null }; secret: ReturnType<typeof ref<string>> }
  let initial!: unknown
  const wrapper = mount(defineComponent({
    setup() {
      state = { search: ref(''), filters: reactive({ status: '', group: null as number | null }), secret: ref('not-a-filter') }
      usePersistedFilters(page, { search: state.search, filters: state.filters }, { skipRestore: () => skipRestore })
      onMounted(() => { initial = { search: state.search.value, ...state.filters } })
      return () => null
    }
  }))
  return { wrapper, state, initial }
}

describe('usePersistedFilters', () => {
  beforeEach(() => { localStorage.clear(); localStorage.setItem('auth_user', JSON.stringify({ id: 1 })) })
  afterEach(() => { vi.restoreAllMocks() })

  it('restores before initial loading and saves only registered filters', () => {
    const first = harness()
    first.state.search.value = 'haiku'
    first.state.filters.status = 'active'
    first.state.filters.group = 7
    first.wrapper.unmount()
    const saved = JSON.parse(localStorage.getItem(pageFilterStorageKey('admin/accounts'))!)
    expect(saved).toEqual({ search: 'haiku', filters: { status: 'active', group: 7 } })
    const second = harness()
    expect(second.initial).toEqual({ search: 'haiku', status: 'active', group: 7 })
  })

  it('separates pages and accounts', () => {
    const first = harness()
    first.state.search.value = 'private filter'
    first.wrapper.unmount()
    expect(harness('admin/users').state.search.value).toBe('')
    localStorage.setItem('auth_user', JSON.stringify({ id: 2 }))
    expect(harness().state.search.value).toBe('')
    localStorage.setItem('auth_user', JSON.stringify({ id: 1 }))
    expect(harness().state.search.value).toBe('private filter')
  })

  it('persists a reset instead of resurrecting old filters', () => {
    const first = harness()
    first.state.search.value = 'old'
    first.state.filters.status = 'active'
    first.state.search.value = ''
    first.state.filters.status = ''
    first.wrapper.unmount()
    expect(harness().initial).toEqual({ search: '', status: '', group: null })
  })

  it('lets explicit link filters override history', () => {
    localStorage.setItem(pageFilterStorageKey('admin/accounts'), JSON.stringify({ search: 'old' }))
    const current = harness('admin/accounts', true)
    expect(current.state.search.value).toBe('')
    current.state.search.value = 'from link'
    current.wrapper.unmount()
    expect(harness().state.search.value).toBe('from link')
  })

  it('ignores corrupt, incompatible and prototype keys', () => {
    const key = pageFilterStorageKey('admin/accounts')
    localStorage.setItem(key, 'not-json')
    expect(() => harness()).not.toThrow()
    localStorage.setItem(key, '{"search":{},"filters":{"status":[],"__proto__":{"polluted":true}}}')
    expect(harness().initial).toEqual({ search: '', status: '', group: null })
    expect(({} as Record<string, unknown>).polluted).toBeUndefined()
  })

  it('keeps working when storage is unavailable', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new Error('denied') })
    expect(() => harness()).not.toThrow()
  })
})
