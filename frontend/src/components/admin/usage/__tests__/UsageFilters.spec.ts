import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

import UsageFilters from '../UsageFilters.vue'

// --- i18n messages (only what UsageFilters needs) ---
const messages: Record<string, string> = {
  'admin.usage.userDeletedBadge': 'deleted',
  'admin.usage.userFilter': 'User',
  'admin.usage.searchUserPlaceholder': 'Search user...',
  'usage.apiKeyFilter': 'API Key',
  'admin.usage.searchApiKeyPlaceholder': 'Search API key...',
  'usage.model': 'Model',
  'admin.usage.allModels': 'All Models',
  'admin.usage.account': 'Account',
  'admin.usage.searchAccountPlaceholder': 'Search account...',
  'usage.type': 'Type',
  'admin.usage.allTypes': 'All Types',
  'usage.ws': 'WS',
  'usage.stream': 'Stream',
  'usage.sync': 'Sync',
  'usage.compactionFilter': 'Request Kind',
  'usage.allCompactionTypes': 'All Requests',
  'usage.compactionOnly': 'Compaction Only',
  'admin.usage.billingType': 'Billing Type',
  'admin.usage.allBillingTypes': 'All Billing Types',
  'admin.usage.billingTypeBalance': 'Balance',
  'admin.usage.billingTypeSubscription': 'Subscription',
  'admin.usage.billingMode': 'Billing Mode',
  'admin.usage.allBillingModes': 'All Billing Modes',
  'admin.usage.billingModeToken': 'Token',
  'admin.usage.billingModePerRequest': 'Per Request',
  'admin.usage.billingModeImage': 'Image',
	'admin.usage.upstreamModelAudit': 'Upstream model audit',
	'admin.usage.allUpstreamModelAudit': 'All response model states',
	'admin.usage.upstreamModelMismatchOnly': 'Mismatched only',
	'admin.usage.upstreamModelMatchedOnly': 'Matched only',
  'admin.usage.group': 'Group',
  'admin.usage.allGroups': 'All Groups',
  'common.refresh': 'Refresh',
  'common.reset': 'Reset',
  'admin.usage.cleanup.button': 'Cleanup',
  'usage.exportExcel': 'Export',
}

// Mock vue-i18n
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

// Mock the admin API module — we control searchUsers return value per test
const mockSearchUsers = vi.fn()
const mockSearchApiKeys = vi.fn().mockResolvedValue([])
const mockGroupsList = vi.fn().mockResolvedValue({ items: [] })
const mockGetModelStats = vi.fn().mockResolvedValue({ models: [] })
const mockAccountsList = vi.fn().mockResolvedValue({ items: [] })

vi.mock('@/api/admin', () => ({
  adminAPI: {
    usage: {
      searchUsers: (...args: any[]) => mockSearchUsers(...args),
      searchApiKeys: (...args: any[]) => mockSearchApiKeys(...args),
    },
    groups: { list: (...args: any[]) => mockGroupsList(...args) },
    dashboard: { getModelStats: (...args: any[]) => mockGetModelStats(...args) },
    accounts: { list: (...args: any[]) => mockAccountsList(...args) },
  },
}))

// Default props helper
const defaultFilters = () => ({
  user_id: undefined,
  api_key_id: undefined,
  account_id: undefined,
  model: null,
  request_type: null,
  native_compaction_v2: null,
  billing_type: null,
  billing_mode: null,
	upstream_model_mismatch: null,
  group_id: null,
  start_date: '',
  end_date: '',
})

function mountFilters(filters = defaultFilters()) {
  return mount(UsageFilters, {
    props: {
      modelValue: filters,
      exporting: false,
      startDate: '2026-05-01',
      endDate: '2026-05-28',
      showActions: false,
      modelOptions: [],
    },
    global: {
      stubs: {
        Select: true,
        Teleport: true,
      },
    },
  })
}

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve
    reject = promiseReject
  })

  return { promise, resolve, reject }
}

describe('UsageFilters — searchable dropdowns', () => {
  beforeEach(() => {
    mockSearchUsers.mockReset().mockResolvedValue([
      { id: 2, email: 'deleted@test.com', deleted: true },
      { id: 1, email: 'active@test.com', deleted: false },
    ])
    mockSearchApiKeys.mockReset().mockResolvedValue([{ id: 3, name: 'Key' }])
    mockAccountsList.mockReset().mockResolvedValue({ items: [{ id: 7, name: 'Account' }] })
  })

  it('loads initial choices without typing and selecting a user clears the old key', async () => {
    const wrapper = mountFilters()
    await flushPromises()
    expect(mockSearchUsers).toHaveBeenCalledWith('')
    expect(mockAccountsList).toHaveBeenCalledWith(1, 1000, { search: '' })
    const selects = wrapper.findAllComponents({ name: 'Select' })
    const user = selects[0]
    expect(user.props('options').map((o: any) => o.value)).toEqual([null, 1, 2])
    expect(user.props('options')[2].label).toContain('deleted')
    wrapper.props('modelValue').api_key_id = 3 as any
    user.vm.$emit('change', 1, { value: 1, label: 'active@test.com' })
    await flushPromises()
    expect(wrapper.props('modelValue').user_id).toBe(1)
    expect(wrapper.props('modelValue').api_key_id).toBeUndefined()
    expect(mockSearchApiKeys).toHaveBeenLastCalledWith(1, '')
    const account = selects[3]
    account.vm.$emit('change', 7, { value: 7, label: 'Account' })
    expect(wrapper.props('modelValue').account_id).toBe(7)
    account.vm.$emit('change', null, null)
    expect(wrapper.props('modelValue').account_id).toBeUndefined()
    wrapper.unmount()
  })

  it('keeps the latest remote search results when requests finish out of order', async () => {
    const wrapper = mountFilters()
    await flushPromises()
    const first = deferred<any[]>()
    const second = deferred<any[]>()
    mockSearchUsers.mockImplementationOnce(() => first.promise).mockImplementationOnce(() => second.promise)
    const user = wrapper.findAllComponents({ name: 'Select' })[0]
    user.vm.$emit('search', 'old')
    user.vm.$emit('search', 'new')
    second.resolve([{ id: 8, email: 'new@test.com', deleted: false }])
    await flushPromises()
    first.resolve([{ id: 9, email: 'old@test.com', deleted: false }])
    await flushPromises()
    expect(user.props('options').map((o: any) => o.value)).toEqual([null, 8])
    wrapper.unmount()
  })
})

describe('UsageFilters — model options come from prop (no dup request)', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockGetModelStats.mockClear()
    mockGroupsList.mockClear()
  })
  afterEach(() => { vi.useRealTimers() })

  it('does not call dashboard.getModelStats on mount and renders model options from prop', async () => {
    const wrapper = mount(UsageFilters, {
      props: {
        modelValue: defaultFilters(),
        exporting: false,
        startDate: '2026-05-01',
        endDate: '2026-05-28',
        showActions: false,
        modelOptions: ['claude-3', 'gpt-4o'],
      },
      global: { stubs: { Select: true, Teleport: true } },
    })
    await flushPromises()

    expect(mockGetModelStats).not.toHaveBeenCalled()

    const opts = (wrapper.vm as any).modelOptions as Array<{ value: string | null; label: string }>
    expect(opts.map((o) => o.value)).toEqual([null, 'claude-3', 'gpt-4o'])
  })
})

describe('UsageFilters — native compaction filter', () => {
  it('offers only All/Compaction and emits the independent boolean filter', async () => {
    const SelectStub = {
      name: 'Select',
      props: ['modelValue', 'options'],
      emits: ['update:modelValue', 'change'],
      template: '<div />',
    }
    const filters = defaultFilters()
    const wrapper = mount(UsageFilters, {
      props: {
        modelValue: filters,
        exporting: false,
        startDate: '2026-05-01',
        endDate: '2026-05-28',
        showActions: false,
        modelOptions: [],
      },
      global: { stubs: { Select: SelectStub, Teleport: true } },
    })

    const compactionSelect = wrapper.findAllComponents(SelectStub).find((select: any) =>
      (select.props('options') as Array<{ value: unknown }>).some((option) => option.value === true)
    )
    expect(compactionSelect).toBeDefined()
    expect(compactionSelect!.props('options')).toEqual([
      { value: null, label: 'All Requests' },
      { value: true, label: 'Compaction Only' },
    ])
    expect(compactionSelect!.props('options')).not.toEqual(
      expect.arrayContaining([expect.objectContaining({ value: false })])
    )

    compactionSelect!.vm.$emit('update:modelValue', true)
    compactionSelect!.vm.$emit('change')
    await wrapper.vm.$nextTick()

    expect(filters.native_compaction_v2).toBe(true)
    expect(wrapper.emitted('change')).toBeTruthy()
  })
})

describe('UsageFilters minimum TTFT', () => {
  it('converts seconds to milliseconds and clears the threshold', async () => {
    const filters: Record<string, any> = defaultFilters()
    const wrapper = mountFilters(filters)
    await flushPromises()
    await wrapper.get('#min-first-token').setValue('2.5')
    expect(filters.min_first_token_ms).toBe(2500)
    expect(wrapper.emitted('change')).toBeTruthy()
    await wrapper.get('#min-first-token').setValue('0')
    expect(filters.min_first_token_ms).toBe(0)
    await wrapper.get('#min-first-token').setValue('')
    expect(filters.min_first_token_ms).toBeUndefined()
    wrapper.unmount()
  })
})
