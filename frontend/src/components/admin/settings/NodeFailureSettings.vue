<template>
  <section class="card" data-testid="node-failure-settings">
    <div class="border-b border-gray-100 px-4 py-4 sm:px-6 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.nodeFailure.title') }}</h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.settings.nodeFailure.description') }}</p>
    </div>
    <div class="space-y-4 p-4 sm:p-6">
      <p v-if="loading" role="status">{{ t('common.loading') }}</p>
      <div v-else-if="loadFailed" role="alert">
        <p class="text-sm text-red-600">{{ t('admin.settings.nodeFailure.loadFailed') }}</p>
        <button type="button" class="btn btn-secondary mt-2" @click="load">{{ t('admin.settings.nodeFailure.retry') }}</button>
      </div>
      <template v-else>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <label v-for="field in fields" :key="field.key" class="block text-sm font-medium">
            {{ t(`admin.settings.nodeFailure.${field.key}`) }}
            <input v-model.number="form[field.key]" :data-testid="field.key" type="number" min="1" :max="field.max" step="1" :disabled="saving" class="input mt-2 w-full" @keydown.enter.prevent="save" />
            <span class="mt-1 block text-xs font-normal text-gray-500">1–{{ field.max }}</span>
          </label>
        </div>
        <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.settings.nodeFailure.recoveryHint') }}</p>
        <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
        <p v-if="saved" role="status" class="text-sm text-green-600">{{ t('admin.settings.nodeFailure.saved') }}</p>
        <div class="flex justify-end">
          <button type="button" class="btn btn-primary w-full sm:w-auto" :disabled="saving" @click="save">{{ saving ? t('common.saving') : t('common.save') }}</button>
        </div>
      </template>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getNodeFailureSettings, updateNodeFailureSettings, type NodeFailureSettings } from '@/api/admin/settings'

const { t } = useI18n()
const loading = ref(true)
const loadFailed = ref(false)
const saving = ref(false)
const saved = ref(false)
const error = ref('')
const form = reactive<NodeFailureSettings>({ window_minutes: 1, failure_threshold: 100, cooldown_minutes: 3, failure_rate_percent: 40 })
const fields: { key: keyof NodeFailureSettings; max: number }[] = [
  { key: 'window_minutes', max: 60 },
  { key: 'failure_threshold', max: 100000 },
  { key: 'failure_rate_percent', max: 99 },
  { key: 'cooldown_minutes', max: 1440 },
]
async function load() {
  loading.value = true
  loadFailed.value = false
  try { Object.assign(form, await getNodeFailureSettings()) }
  catch { loadFailed.value = true }
  finally { loading.value = false }
}
async function save() {
  if (saving.value || loading.value || loadFailed.value) return
  saved.value = false
  error.value = ''
  if (fields.some(({ key, max }) => !Number.isInteger(form[key]) || form[key] < 1 || form[key] > max)) {
    error.value = t('admin.settings.nodeFailure.invalid')
    return
  }
  saving.value = true
  try {
    Object.assign(form, await updateNodeFailureSettings({ ...form }))
    saved.value = true
  } catch { error.value = t('admin.settings.nodeFailure.saveFailed') }
  finally { saving.value = false }
}
onMounted(load)
</script>
