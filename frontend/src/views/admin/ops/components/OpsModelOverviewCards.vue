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

// Calculate time window in hours for daily/total display
const timeWindowHours = computed(() => {
  if (props.customStartTime && props.customEndTime) {
    const start = new Date(props.customStartTime).getTime()
    const end = new Date(props.customEndTime).getTime()
    return (end - start) / (1000 * 60 * 60)
  }

  const ranges: Record<string, number> = {
    '5m': 5 / 60,
    '30m': 0.5,
    '1h': 1,
    '6h': 6,
    '24h': 24
  }
  return ranges[props.timeRange] || 1
})

const isToday = computed(() => {
  return props.timeRange === '24h' || timeWindowHours.value <= 24
})

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
    params.top_n = 10 // Show top 10 models in overview

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

function formatTokens(tokens: number): string {
  if (tokens >= 1_000_000_000) {
    return `${(tokens / 1_000_000_000).toFixed(2)}B`
  } else if (tokens >= 1_000_000) {
    return `${(tokens / 1_000_000).toFixed(2)}M`
  } else if (tokens >= 1_000) {
    return `${(tokens / 1_000).toFixed(1)}K`
  }
  return formatNumber(tokens)
}

function formatCurrency(amount: number): string {
  if (amount >= 1000) {
    return `$${(amount / 1000).toFixed(2)}K`
  }
  return `$${amount.toFixed(2)}`
}

function formatDuration(ms: number | null): string {
  if (ms === null || ms === undefined) return '-'
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(2)}s`
  }
  return `${ms.toFixed(0)}ms`
}

function formatRPM(requests: number): string {
  const minutes = timeWindowHours.value * 60
  if (minutes === 0) return '0'
  const rpm = requests / minutes
  return formatNumber(Math.round(rpm))
}

function formatTPM(tokens: number): string {
  const minutes = timeWindowHours.value * 60
  if (minutes === 0) return '0'
  const tpm = tokens / minutes
  if (tpm >= 1_000_000) {
    return `${(tpm / 1_000_000).toFixed(2)}M`
  } else if (tpm >= 1_000) {
    return `${(tpm / 1_000).toFixed(1)}K`
  }
  return formatNumber(Math.round(tpm))
}

// Mock cost calculation (should come from backend in production)
function estimateCost(tokens: number): { input: number; output: number; total: number } {
  // Simplified cost estimation - in production this should use actual pricing
  const inputCost = tokens * 0.00001 * 0.4  // 40% input
  const outputCost = tokens * 0.00003 * 0.6 // 60% output
  return {
    input: inputCost,
    output: outputCost,
    total: inputCost + outputCost
  }
}

function getErrorRateClass(rate: number): string {
  if (rate >= 0.1) return 'text-red-600 dark:text-red-400'
  if (rate >= 0.05) return 'text-orange-600 dark:text-orange-400'
  if (rate >= 0.01) return 'text-yellow-600 dark:text-yellow-400'
  return 'text-green-600 dark:text-green-400'
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
  <div class="ops-model-overview">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ t('admin.ops.modelOverview.title') }}
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
      {{ t('admin.ops.modelOverview.noData') }}
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
      <div
        v-for="model in modelMetrics"
        :key="model.model"
        class="bg-white dark:bg-gray-800 rounded-lg shadow p-4 border border-gray-200 dark:border-gray-700 hover:shadow-md transition-shadow"
      >
        <!-- Model Name Header -->
        <div class="flex items-center justify-between mb-3 pb-2 border-b border-gray-200 dark:border-gray-700">
          <h4 class="font-semibold text-gray-900 dark:text-white truncate" :title="model.model">
            {{ model.model }}
          </h4>
          <span class="text-xs px-2 py-1 rounded" :class="getErrorRateClass(model.error_rate)">
            {{ (model.error_rate * 100).toFixed(1) }}%
          </span>
        </div>

        <!-- Token Stats -->
        <div class="space-y-2 mb-3">
          <div class="flex justify-between items-baseline">
            <span class="text-xs text-gray-500 dark:text-gray-400">
              {{ isToday ? t('admin.ops.modelOverview.todayTokens') : t('admin.ops.modelOverview.tokens') }}
            </span>
            <span class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ formatTokens(model.total_tokens) }}
            </span>
          </div>
          <div class="flex justify-between items-baseline">
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.ops.modelOverview.cost') }}</span>
            <span class="text-xs text-gray-700 dark:text-gray-300">
              {{ formatCurrency(estimateCost(model.total_tokens).total) }}
            </span>
          </div>
        </div>

        <!-- Performance Metrics -->
        <div class="space-y-2 mb-3 pt-2 border-t border-gray-100 dark:border-gray-700">
          <div class="flex justify-between items-baseline">
            <span class="text-xs text-gray-500 dark:text-gray-400">RPM</span>
            <span class="text-sm font-medium text-gray-900 dark:text-white">
              {{ formatRPM(model.request_count) }}
            </span>
          </div>
          <div class="flex justify-between items-baseline">
            <span class="text-xs text-gray-500 dark:text-gray-400">TPM</span>
            <span class="text-sm font-medium text-gray-900 dark:text-white">
              {{ formatTPM(model.total_tokens) }}
            </span>
          </div>
          <div class="flex justify-between items-baseline">
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.ops.modelOverview.avgResponse') }}</span>
            <span class="text-sm font-medium text-gray-900 dark:text-white">
              {{ formatDuration(model.avg_duration_ms) }}
            </span>
          </div>
        </div>

        <!-- Request Stats -->
        <div class="pt-2 border-t border-gray-100 dark:border-gray-700">
          <div class="flex justify-between items-baseline text-xs">
            <span class="text-gray-500 dark:text-gray-400">{{ t('admin.ops.modelOverview.requests') }}</span>
            <span class="font-medium text-gray-900 dark:text-white">
              {{ formatNumber(model.request_count) }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.ops-model-overview {
  @apply bg-gray-50 dark:bg-gray-900/50 rounded-lg p-6;
}
</style>
