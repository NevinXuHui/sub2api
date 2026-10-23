<template>
  <section class="card" data-testid="account-daily-discount">
    <div class="border-b border-gray-100 px-4 py-4 sm:px-6 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.accountDailyDiscount.title') }}</h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.settings.accountDailyDiscount.description') }}</p>
    </div>
    <div class="space-y-4 p-4 sm:p-6">
      <p v-if="loading" role="status">{{ t('common.loading') }}</p>
      <div v-else-if="loadFailed" role="alert">
        <p class="text-sm text-red-600">{{ t('admin.settings.accountDailyDiscount.loadFailed') }}</p>
        <button type="button" class="btn btn-secondary mt-2" @click="load">{{ t('admin.settings.accountDailyDiscount.retry') }}</button>
      </div>
      <template v-else>
        <label class="flex items-center gap-2 text-sm font-medium">
          <input v-model="form.enabled" type="checkbox" :disabled="saving" data-testid="discount-enabled" />
          {{ t('admin.settings.accountDailyDiscount.enabled') }}
        </label>
        <div v-for="(tier, index) in form.tiers" :key="index" class="grid grid-cols-1 items-end gap-3 rounded-lg border border-gray-100 p-3 sm:grid-cols-[1fr_1fr_auto] dark:border-dark-700">
          <label class="text-sm">
            {{ t('admin.settings.accountDailyDiscount.threshold') }}
            <input v-model.number="tier.threshold_usd" type="number" min="0" max="1000000000" step="0.00000001" :disabled="saving" class="input mt-1 w-full" :data-testid="`threshold-${index}`" @keydown.enter.prevent="save" />
          </label>
          <label class="text-sm">
            {{ t('admin.settings.accountDailyDiscount.discount') }}
            <input v-model.number="tier.discount" type="number" min="0.01" max="10" step="0.01" :disabled="saving" class="input mt-1 w-full" :data-testid="`discount-${index}`" @keydown.enter.prevent="save" />
          </label>
          <button type="button" class="btn btn-secondary" :disabled="saving" :data-testid="`remove-${index}`" @click="form.tiers.splice(index, 1)">{{ t('admin.settings.accountDailyDiscount.remove') }}</button>
        </div>
        <p v-if="!form.tiers.length" class="text-sm text-gray-500">{{ t('admin.settings.accountDailyDiscount.empty') }}</p>
        <button type="button" class="btn btn-secondary" :disabled="saving || form.tiers.length >= 20" data-testid="add-tier" @click="form.tiers.push({ threshold_usd: 0, discount: 10 })">{{ t('admin.settings.accountDailyDiscount.add') }}</button>
        <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.settings.accountDailyDiscount.hint') }}</p>
        <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
        <p v-if="saved" role="status" class="text-sm text-green-600">{{ t('admin.settings.accountDailyDiscount.saved') }}</p>
        <div class="flex justify-end"><button type="button" class="btn btn-primary w-full sm:w-auto" :disabled="saving" data-testid="save-discount" @click="save">{{ saving ? t('common.saving') : t('common.save') }}</button></div>
      </template>
    </div>
  </section>
</template>
<script setup lang="ts">
import { onMounted, reactive, ref, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { getAccountDailyDiscountSettings, updateAccountDailyDiscountSettings, type AccountDailyDiscountSettings } from '@/api/admin/settings'
const { t } = useI18n()
const form = reactive<AccountDailyDiscountSettings>({ enabled: false, tiers: [] })
const loading = ref(true)
const loadFailed = ref(false)
const saving = ref(false)
const saved = ref(false)
const error = ref('')
watch(form, () => { saved.value = false })
async function load() {
  loading.value = true
  loadFailed.value = false
  try { Object.assign(form, await getAccountDailyDiscountSettings()) }
  catch { loadFailed.value = true }
  finally { loading.value = false }
}
function valid() {
  return !(form.enabled && !form.tiers.length) && form.tiers.length <= 20 &&
    new Set(form.tiers.map(tier => tier.threshold_usd)).size === form.tiers.length &&
    form.tiers.every(tier => typeof tier.threshold_usd === 'number' && Number.isFinite(tier.threshold_usd) && tier.threshold_usd >= 0 && tier.threshold_usd <= 1e9 && Number(tier.threshold_usd.toFixed(8)) === tier.threshold_usd &&
      typeof tier.discount === 'number' && Number.isFinite(tier.discount) && tier.discount > 0 && tier.discount <= 10 && Number(tier.discount.toFixed(2)) === tier.discount)
}
async function save() {
  if (saving.value || loading.value || loadFailed.value) return
  saved.value = false
  error.value = ''
  if (!valid()) { error.value = t('admin.settings.accountDailyDiscount.invalid'); return }
  saving.value = true
  try {
    Object.assign(form, await updateAccountDailyDiscountSettings({ enabled: form.enabled, tiers: form.tiers.map(tier => ({ ...tier })) }))
    // Let the dirty-state watcher settle before showing the saved notice.
    await nextTick()
    saved.value = true
  } catch { error.value = t('admin.settings.accountDailyDiscount.saveFailed') }
  finally { saving.value = false }
}
onMounted(load)
</script>
