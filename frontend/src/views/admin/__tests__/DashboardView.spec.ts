import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

import type { DashboardStats } from '@/types'
import DashboardView from '../DashboardView.vue'

const { getSnapshotV2, getUserUsageTrend, getUserSpendingRanking, getPerformance, getModelStats } = vi.hoisted(() => ({
  getPerformance: vi.fn(),
  getModelStats: vi.fn(),
  getSnapshotV2: vi.fn(),
  getUserUsageTrend: vi.fn(),
  getUserSpendingRanking: vi.fn()
}))

vi.mock('vue-chartjs', () => ({
  Line: { name: 'Line', props: ['data', 'options'], template: '<div class="line-chart" />' }
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    dashboard: {
      getPerformance,
      getModelStats,
      getSnapshotV2,
      getUserUsageTrend,
      getUserSpendingRanking
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn()
  })
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const formatLocalDate = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const createDashboardStats = (): DashboardStats => ({
  total_users: 0,
  today_new_users: 0,
  active_users: 0,
  hourly_active_users: 0,
  stats_updated_at: '',
  stats_stale: false,
  total_api_keys: 0,
  active_api_keys: 0,
  total_accounts: 0,
  normal_accounts: 0,
  error_accounts: 0,
  ratelimit_accounts: 0,
  overload_accounts: 0,
  total_requests: 0,
  total_input_tokens: 0,
  total_output_tokens: 0,
  total_cache_creation_tokens: 0,
  total_cache_read_tokens: 0,
  total_tokens: 0,
  total_cost: 0,
  total_actual_cost: 0,
  today_requests: 0,
  today_input_tokens: 0,
  today_output_tokens: 0,
  today_cache_creation_tokens: 0,
  today_cache_read_tokens: 0,
  today_tokens: 0,
  today_cost: 0,
  today_actual_cost: 0,
  average_duration_ms: 0,
  uptime: 0,
  rpm: 0,
  tpm: 0
})

enableAutoUnmount(afterEach)

describe('admin DashboardView', () => {
  it('distinguishes zero available kiro accounts from unavailable instances', async () => {
    getPerformance.mockResolvedValue({ models: [], kiro_rs_accounts: [
      { name: 'kiro-zero', priority_zero_available: 0, status: 'ok' },
      { name: 'kiro-offline', priority_zero_available: null, status: 'unavailable' }
    ] })
    const wrapper = mount(DashboardView, { global: { stubs: {
      AppLayout: { template: '<div><slot /></div>' }, LoadingSpinner: true,
      Icon: true, DateRangePicker: true, Select: true, ModelDistributionChart: true, TokenUsageTrend: true
    } } })
    await flushPromises()
    const rows = wrapper.get('[data-testid="kiro-rs-accounts"]')
    expect(rows.text()).toContain('kiro-zero')
    expect(rows.findAll('strong')[0]?.text()).toBe('0')
    expect(rows.findAll('strong')[1]?.text()).toBe('admin.dashboard.kiroUnavailable')
  })
  afterEach(() => { vi.useRealTimers() })
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())

    getModelStats.mockReset().mockResolvedValue({ models: [] })
    getPerformance.mockReset()
    getPerformance.mockResolvedValue({ error_count: 12, hour_error_count: 60, avg_first_token_ms: 1200, first_token_samples: 100, rpm: 9, tpm: 1690, window_seconds: 60, updated_at: new Date().toISOString(), models: [{ model: 'haiku', rpm: 7, tpm: 1234, avg_first_token_ms: 1500, first_token_samples: 60 }, { model: 'sonnet', rpm: 2, tpm: 456 }] })
    getSnapshotV2.mockReset()
    getUserUsageTrend.mockReset()
    getUserSpendingRanking.mockReset()

    getSnapshotV2.mockResolvedValue({
      stats: createDashboardStats(),
      trend: [],
      models: []
    })
    getUserUsageTrend.mockResolvedValue({
      trend: [],
      start_date: '',
      end_date: '',
      granularity: 'hour'
    })
    getUserSpendingRanking.mockResolvedValue({
      ranking: [],
      total_actual_cost: 0,
      total_requests: 0,
      total_tokens: 0,
      start_date: '',
      end_date: ''
    })
  })

  it('shows today error count and percentage and refreshes them', async () => {
    vi.useFakeTimers()
    getPerformance.mockResolvedValue({ models: [], error_count: 0, today_error_count: 25, today_error_rate: 12.5, today_models: [{ model: "error-only-model", requests: 0, error_count: 25, error_rate: 100 }] })
    const wrapper = mount(DashboardView, { global: { stubs: {
      AppLayout: { template: '<div><slot /></div>' }, LoadingSpinner: true,
      Icon: true, DateRangePicker: true, Select: true, ModelDistributionChart: true, TokenUsageTrend: true
    } } })
    try {
      await flushPromises()
      expect(wrapper.get('[data-testid="today-errors"]').text()).toContain('25')
      expect(wrapper.get('[data-testid="today-errors"]').text()).toContain('12.50%')
      expect(wrapper.get('[data-testid="today-requests-models"]').text()).toContain('error-only-model')
      expect(wrapper.get('[data-testid="today-requests-models"]').text()).toContain('100.00%')
      getPerformance.mockResolvedValue({ models: [], error_count: 0, today_error_count: 0, today_error_rate: 0 })
      await vi.advanceTimersByTimeAsync(10_000)
      await flushPromises()
      expect(wrapper.get('[data-testid="today-errors"]').text()).toContain('0.00%')
    } finally {
      wrapper.unmount()
      vi.useRealTimers()
    }
  })

  it('filters the token trend by model and restores the selection after remount', async () => {
    const options = { global: { stubs: {
      AppLayout: { template: '<div><slot /></div>' }, LoadingSpinner: true,
      Icon: true, DateRangePicker: true, Select: true, ModelDistributionChart: true, TokenUsageTrend: true
    } } }
    const wrapper = mount(DashboardView, options)
    await flushPromises()
    wrapper.findComponent({ name: 'TokenUsageTrend' }).vm.$emit('update:model', 'claude-sonnet-4-5')
    await flushPromises()
    expect(getSnapshotV2).toHaveBeenLastCalledWith(expect.objectContaining({ model: 'claude-sonnet-4-5' }))
    wrapper.unmount()
    const restored = mount(DashboardView, options)
    await flushPromises()
    expect(restored.findComponent({ name: 'TokenUsageTrend' }).props('model')).toBe('claude-sonnet-4-5')
    expect(getSnapshotV2).toHaveBeenLastCalledWith(expect.objectContaining({ model: 'claude-sonnet-4-5' }))
    restored.findComponent({ name: 'TokenUsageTrend' }).vm.$emit('update:model', '')
    await flushPromises()
    expect(getSnapshotV2).toHaveBeenLastCalledWith(expect.objectContaining({ model: undefined }))
  })

  it('shows independent daily and lifetime model totals and refreshes each minute', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 9, 17, 12))
    getModelStats.mockImplementation(async ({ start_date }) => ({ models: start_date === '1970-01-01'
      ? [{ model: 'old-model', requests: 9, total_tokens: 900, actual_cost: 12, account_cost: 6, cost: 24 }]
      : [{ model: 'haiku', requests: 3, total_tokens: 100, actual_cost: 2, account_cost: 1, cost: 4 }, { model: 'sonnet', requests: 2, total_tokens: 200, actual_cost: 3, account_cost: 2, cost: 6 }] }))
    const wrapper = mount(DashboardView, { global: { stubs: {
      AppLayout: { template: '<div><slot /></div>' }, LoadingSpinner: true,
      Icon: true, DateRangePicker: true, Select: true, ModelDistributionChart: true, TokenUsageTrend: true
    } } })
    await flushPromises()
    expect(getModelStats).toHaveBeenCalledWith({ start_date: '2026-10-17', end_date: '2026-10-17', model_source: 'requested' }, expect.any(AbortSignal))
    expect(getModelStats).toHaveBeenCalledWith({ start_date: '1970-01-01', end_date: '2026-10-17', model_source: 'requested' }, expect.any(AbortSignal))
    expect(wrapper.get('[data-testid="today-requests-models"]').text()).toContain('haiku')
    expect(wrapper.get('[data-testid="today-total_tokens-models"]').text()).toContain('sonnet')
    expect(wrapper.get('[data-testid="total-total_tokens-models"]').text()).toContain('old-model')
    expect(wrapper.get('[data-testid="today-total_tokens-models"]').text()).not.toContain('old-model')
    expect((wrapper.vm as any).summaryTotals).toEqual({ todayRequests: 5, todayTokens: 300, totalTokens: 900 })
    expect((wrapper.vm as any).summaryCosts).toEqual({ today: { actual_cost: 5, account_cost: 3, cost: 10 }, total: { actual_cost: 12, account_cost: 6, cost: 24 } })
    const costs = wrapper.findAll('[data-testid="today-model-costs"]')
    expect(costs).toHaveLength(2)
    expect(costs[0].text()).toContain('$2.00')
    expect(costs[0].text()).toContain('$1.00')
    expect(costs[0].text()).toContain('$4.00')
    expect(wrapper.get('[data-testid="total-model-costs"]').text()).toContain('$12.00')
    getModelStats.mockRejectedValueOnce(new Error('offline'))
    await vi.advanceTimersByTimeAsync(60_000)
    await flushPromises()
    expect(getModelStats).toHaveBeenCalledTimes(4)
    expect(wrapper.get('[data-testid="total-total_tokens-models"]').text()).toContain('old-model')
    expect(wrapper.get('[data-testid="total-total_tokens-models"]').text()).toContain('admin.dashboard.failedToLoad')
    wrapper.unmount()
    await vi.advanceTimersByTimeAsync(60_000)
    expect(getModelStats).toHaveBeenCalledTimes(4)
  })

  it('shows model rates, refreshes every ten seconds, and stops on unmount', async () => {
    vi.useFakeTimers()
    getPerformance.mockResolvedValueOnce({ ...await getPerformance(), error_free_seconds: 176461 })
    getPerformance.mockClear()
    const wrapper = mount(DashboardView, {
      global: { stubs: {
        AppLayout: { template: '<div><slot /></div>' }, LoadingSpinner: true,
        Icon: true, DateRangePicker: true, Select: true, ModelDistributionChart: true,
        TokenUsageTrend: true
      } }
    })
    await flushPromises()
    const table = wrapper.get('[data-testid="model-performance"]')
    expect(table.text()).toContain('haiku')
    expect(table.text()).toContain('sonnet')
    expect(table.text()).toContain('1.50 s')
    expect(wrapper.get('[data-testid="recent-ttft"]').text()).toContain('1.20 s')
    expect(getPerformance).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-testid="recent-errors"]').text()).toContain('12')
    expect(wrapper.get('[data-testid="hour-errors"]').text()).toContain('60')
    expect(wrapper.get('[data-testid="error-free-time"]').text()).toContain('01:01:01')
    expect(wrapper.get('[data-testid="error-free-time"]').text()).toContain('admin.dashboard.errorFreeDays')
    await vi.advanceTimersByTimeAsync(1000)
    expect(wrapper.get('[data-testid="error-free-time"]').text()).toContain('01:01:02')
    expect(getPerformance).toHaveBeenCalledTimes(1)
    getPerformance.mockRejectedValueOnce(new Error('offline'))
    await vi.advanceTimersByTimeAsync(9000)
    await flushPromises()
    expect(getPerformance).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('admin.dashboard.performanceRefreshFailed')
    expect(table.text()).toContain('haiku')
    expect(wrapper.get('[data-testid="recent-errors"]').text()).toContain('12')
    expect(wrapper.get('[data-testid="hour-errors"]').text()).toContain('60')
    getPerformance.mockResolvedValueOnce({ error_free_seconds: null, error_count: 0, hour_error_count: 0, rpm: 0, tpm: 0, window_seconds: 60, updated_at: new Date().toISOString(), models: [] })
    await vi.advanceTimersByTimeAsync(10_000)
    await flushPromises()
    expect(table.text()).not.toContain('haiku')
    expect(wrapper.get('[data-testid="recent-errors"]').text()).toContain('0')
    expect(wrapper.get('[data-testid="hour-errors"]').text()).toContain('0')
    expect(wrapper.get('[data-testid="error-free-time"]').text()).toContain('admin.dashboard.noErrorHistory')
    expect(wrapper.text()).not.toContain('admin.dashboard.performanceRefreshFailed')
    wrapper.unmount()
    await vi.advanceTimersByTimeAsync(10_000)
    expect(getPerformance).toHaveBeenCalledTimes(3)
  })

  it('switches metrics and ignores a stale tokens response', async () => {
    let resolveTokens!: (value: any) => void
    getUserUsageTrend.mockImplementationOnce(() => new Promise(resolve => { resolveTokens = resolve }))
    getUserUsageTrend.mockResolvedValueOnce({ trend: [{ date: '2026-01-01', user_id: 2, username: 'spender', email: '', requests: 1, tokens: 10, cost: 5, actual_cost: 5 }] })
    const wrapper = mount(DashboardView, {
      global: { stubs: {
        AppLayout: { template: '<div><slot /></div>' }, LoadingSpinner: true,
        Icon: true, DateRangePicker: true, Select: true, ModelDistributionChart: true,
        TokenUsageTrend: true
      } }
    })
    await flushPromises()
    expect(getUserUsageTrend).toHaveBeenCalledWith(expect.objectContaining({ metric: 'tokens', limit: 12 }))
    await wrapper.findAll('button').find(button => button.text() === 'admin.dashboard.actualSpending')!.trigger('click')
    expect(getUserUsageTrend).toHaveBeenCalledWith(expect.objectContaining({ metric: 'actual_cost', limit: 12 }))
    await flushPromises()
    const chart = wrapper.findComponent({ name: 'Line' })
    expect(chart.props('data').datasets[0].data).toEqual([5])
    expect(chart.props('options').scales.y.ticks.callback(5)).toBe('$5.00')
    resolveTokens({ trend: [{ date: '2026-01-01', user_id: 1, username: 'tokens', email: '', requests: 1, tokens: 100, cost: 1, actual_cost: 1 }] })
    await flushPromises()
    expect(chart.props('data').datasets[0].data).toEqual([5])
  })

  it('uses last 24 hours as default dashboard range', async () => {
    mount(DashboardView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          LoadingSpinner: true,
          Icon: true,
          DateRangePicker: true,
          Select: true,
          ModelDistributionChart: true,
          TokenUsageTrend: true,
          Line: true
        }
      }
    })

    await flushPromises()

    const now = new Date()
    const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000)

    expect(getSnapshotV2).toHaveBeenCalledTimes(1)
    expect(getSnapshotV2).toHaveBeenCalledWith(expect.objectContaining({
      start_date: formatLocalDate(yesterday),
      end_date: formatLocalDate(now),
      granularity: 'hour'
    }))
  })
})
