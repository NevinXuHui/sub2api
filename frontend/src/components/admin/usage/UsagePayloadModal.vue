<template>
  <BaseDialog :show="show" :title="t('usage.payload.title')" width="wide" @close="emit('update:show', false)">
    <div v-if="loading" class="flex justify-center py-10">
      <svg class="h-7 w-7 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
      </svg>
    </div>

    <div v-else-if="loadError" class="py-8 text-center text-sm text-red-500">
      {{ t('usage.payload.loadFailed') }}
    </div>

    <div v-else-if="detail && !detail.captured" class="py-8 text-center text-sm text-gray-500 dark:text-dark-400">
      {{ t('usage.payload.notCaptured') }}
    </div>

    <div v-else-if="detail" class="space-y-4 text-sm">
      <div>
        <div class="flex items-center justify-between gap-2">
          <span class="font-medium text-gray-500 dark:text-dark-400">{{ t('usage.payload.request') }}</span>
          <span v-if="detail.request_truncated" class="text-xs text-amber-600 dark:text-amber-400">
            {{ t('usage.payload.truncated', { bytes: detail.request_bytes }) }}
          </span>
        </div>
        <pre class="mt-1 max-h-[32vh] overflow-auto whitespace-pre-wrap break-all rounded-lg border border-gray-200 bg-gray-50 p-3 text-xs text-gray-800 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-200">{{ detail.request_body || t('usage.payload.empty') }}</pre>
      </div>
      <div>
        <div class="flex items-center justify-between gap-2">
          <span class="font-medium text-gray-500 dark:text-dark-400">{{ t('usage.payload.response') }}</span>
          <span v-if="detail.response_truncated" class="text-xs text-amber-600 dark:text-amber-400">
            {{ t('usage.payload.truncated', { bytes: detail.response_bytes }) }}
          </span>
        </div>
        <pre class="mt-1 max-h-[32vh] overflow-auto whitespace-pre-wrap break-all rounded-lg border border-gray-200 bg-gray-50 p-3 text-xs text-gray-800 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-200">{{ detail.response_body || t('usage.payload.empty') }}</pre>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { getMyUsagePayload } from '@/api/usage'
import { getAdminUsagePayload } from '@/api/admin/usage'
import type { UsagePayload } from '@/types'

const props = defineProps<{
  show: boolean
  usageId: number | null
  scope: 'admin' | 'user'
}>()

const emit = defineEmits<{
  (e: 'update:show', v: boolean): void
}>()

const { t } = useI18n()
const loading = ref(false)
const loadError = ref(false)
const detail = ref<UsagePayload | null>(null)

watch(
  () => [props.show, props.usageId, props.scope] as const,
  ([show, id]) => {
    if (show && id != null) {
      fetchDetail(id)
    } else if (!show) {
      detail.value = null
      loadError.value = false
    }
  }
)

async function fetchDetail(id: number) {
  loading.value = true
  loadError.value = false
  detail.value = null
  try {
    detail.value = props.scope === 'admin'
      ? await getAdminUsagePayload(id)
      : await getMyUsagePayload(id)
  } catch (e) {
    console.error('[UsagePayloadModal] Failed to load payload:', e)
    loadError.value = true
  } finally {
    loading.value = false
  }
}
</script>
