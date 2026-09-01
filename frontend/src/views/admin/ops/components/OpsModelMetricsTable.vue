<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getModelBreakdown, type OpsModelBreakdownItem } from '@/api/admin/ops'
import { formatNumber } from '@/utils/format'

interface Props {
  platform: string
  groupId: number | null
  timeRange: string
  customStartTime?: string | null
  customEndTime?: string | null
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  loading: false
})

const { t } = useI18n()

const modelMetrics = ref<OpsModelBreakdownItem[]>([])
const internalLoading = ref(false)
const error = ref<string | null>(null)

const isLoading = computed(() => props.loading || internalLoading.value)

async function loadModelMetrics() {
  if (internalLoading.value) return

  internalLoading.value = true
  error.value = null

  try {
    const params: any = {}

    if (props.customStartTime && props.customEndTime) {
      params.start_time = props.customStartTime
      params.end_time = props.customEndTime
    } else {
      params.time_range = props.timeRange
    }

    if (props.platform) {
      params.platform = props.platform
    }
    if (props.groupId !== null) {
      params.group_id = props.groupId
    }
    params.top_n = 20 // Show top 20 models

    const resp = await getModelBreakdown(params)
    modelMetrics.value = resp.items || []
  } catch (err: any) {
    console.error('Failed to load model metrics:', err)
    error.value = err.message || 'Failed to load model metrics'
    modelMetrics.value = []
  } finally {
    internalLoading.value = false
  }
}

function formatMs(ms: number | null): string {
  if (ms === null || ms === undefined) return '-'
  return `${ms.toFixed(0)}ms`
}

function formatRate(rate: number): string {
  return `${(rate * 100).toFixed(2)}%`
}

function formatTokens(tokens: number): string {
  if (tokens >= 1_000_000) {
    return `${(tokens / 1_000_000).toFixed(2)}M`
  } else if (tokens >= 1_000) {
    return `${(tokens / 1_000).toFixed(1)}K`
  }
  return formatNumber(tokens)
}

function formatTPS(tps: number | null): string {
  if (tps === null || tps === undefined) return '-'
  return tps.toFixed(1)
}

function getErrorRateClass(rate: number): string {
  if (rate >= 0.1) return 'text-red-600 dark:text-red-400 font-semibold'
  if (rate >= 0.05) return 'text-orange-600 dark:text-orange-400'
  if (rate >= 0.01) return 'text-yellow-600 dark:text-yellow-400'
  return 'text-green-600 dark:text-green-400'
}

function getLatencyClass(ms: number | null): string {
  if (ms === null) return ''
  if (ms >= 3000) return 'text-red-600 dark:text-red-400'
  if (ms >= 2000) return 'text-orange-600 dark:text-orange-400'
  if (ms >= 1000) return 'text-yellow-600 dark:text-yellow-400'
  return ''
}

onMounted(() => {
  loadModelMetrics()
})

watch(
  () => [props.platform, props.groupId, props.timeRange, props.customStartTime, props.customEndTime],
  () => {
    loadModelMetrics()
  }
)

defineExpose({ refresh: loadModelMetrics })
</script>

<template>
  <div class="ops-model-metrics">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ t('admin.ops.modelMetrics.title') }}
      </h3>
      <button
        v-if="!isLoading"
        @click="loadModelMetrics"
        class="text-sm text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
      >
        🔄 {{ t('common.refresh') }}
      </button>
    </div>

    <div v-if="error" class="p-4 mb-4 bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 rounded">
      {{ error }}
    </div>

    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900 dark:border-white"></div>
    </div>

    <div v-else-if="modelMetrics.length === 0" class="text-center py-12 text-gray-500 dark:text-gray-400">
      {{ t('admin.ops.modelMetrics.noData') }}
    </div>

    <div v-else class="overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
        <thead class="bg-gray-50 dark:bg-gray-800">
          <tr>
            <th class="px-3 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              {{ t('admin.ops.modelMetrics.model') }}
            </th>
            <th class="px-3 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              {{ t('admin.ops.modelMetrics.requests') }}
            </th>
            <th class="px-3 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              {{ t('admin.ops.modelMetrics.errorRate') }}
            </th>
            <th class="px-3 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              {{ t('admin.ops.modelMetrics.tokens') }}
            </th>
            <th class="px-3 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              {{ t('admin.ops.modelMetrics.avgDuration') }}
            </th>
            <th class="px-3 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              P95
            </th>
            <th class="px-3 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              P99
            </th>
            <th class="px-3 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              {{ t('admin.ops.modelMetrics.ttftP99') }}
            </th>
            <th class="px-3 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              {{ t('admin.ops.modelMetrics.qps') }}
            </th>
          </tr>
        </thead>
        <tbody class="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
          <tr v-for="item in modelMetrics" :key="item.model" class="hover:bg-gray-50 dark:hover:bg-gray-800">
            <td class="px-3 py-3 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
              {{ item.model }}
            </td>
            <td class="px-3 py-3 whitespace-nowrap text-sm text-right text-gray-900 dark:text-gray-100">
              {{ formatNumber(item.request_count) }}
            </td>
            <td class="px-3 py-3 whitespace-nowrap text-sm text-right" :class="getErrorRateClass(item.error_rate)">
              {{ formatRate(item.error_rate) }}
            </td>
            <td class="px-3 py-3 whitespace-nowrap text-sm text-right text-gray-700 dark:text-gray-300">
              {{ formatTokens(item.total_tokens) }}
            </td>
            <td class="px-3 py-3 whitespace-nowrap text-sm text-right text-gray-700 dark:text-gray-300">
              {{ formatMs(item.avg_duration_ms) }}
            </td>
            <td class="px-3 py-3 whitespace-nowrap text-sm text-right" :class="getLatencyClass(item.p95_duration_ms)">
              {{ formatMs(item.p95_duration_ms) }}
            </td>
            <td class="px-3 py-3 whitespace-nowrap text-sm text-right" :class="getLatencyClass(item.p99_duration_ms)">
              {{ formatMs(item.p99_duration_ms) }}
            </td>
            <td class="px-3 py-3 whitespace-nowrap text-sm text-right" :class="getLatencyClass(item.p99_ttft_ms)">
              {{ formatMs(item.p99_ttft_ms) }}
            </td>
            <td class="px-3 py-3 whitespace-nowrap text-sm text-right text-gray-700 dark:text-gray-300">
              {{ item.qps.toFixed(2) }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.ops-model-metrics {
  @apply bg-white dark:bg-gray-900 rounded-lg shadow p-6;
}
</style>
