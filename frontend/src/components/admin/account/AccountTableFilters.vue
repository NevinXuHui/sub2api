<template>
  <!-- 窄屏两列网格铺满宽度，sm 起恢复单行流式排布 -->
  <div class="grid w-full grid-cols-2 gap-3 sm:flex sm:w-auto sm:flex-wrap sm:items-center">
    <SearchInput
      :model-value="searchQuery"
      :placeholder="t('admin.accounts.searchAccounts')"
      class="col-span-2 sm:w-52"
      @update:model-value="$emit('update:searchQuery', $event)"
      @search="$emit('change')"
    />
    <Select :model-value="filters.platform" class="min-w-0 sm:w-40" :options="pOpts" @update:model-value="updatePlatform" @change="$emit('change')" />
    <Select :model-value="filters.type" class="min-w-0 sm:w-40" :options="tOpts" @update:model-value="updateType" @change="$emit('change')" />
    <Select :model-value="filters.status" class="min-w-0 sm:w-40" :options="sOpts" @update:model-value="updateStatus" @change="$emit('change')" />
    <Select :model-value="filters.privacy_mode" class="min-w-0 sm:w-48" :options="privacyOpts" @update:model-value="updatePrivacyMode" @change="$emit('change')" />
    <details class="relative col-span-2 min-w-0 sm:w-56">
      <summary class="input cursor-pointer select-none" :title="selectedGroupLabels">{{ selectedGroupLabels || t('admin.accounts.allGroups') }}</summary>
      <div class="absolute left-0 z-50 mt-1 w-full min-w-56 rounded-lg border border-gray-200 bg-white p-2 shadow-lg dark:border-dark-600 dark:bg-dark-800">
        <input v-model="groupSearch" class="input mb-2" :placeholder="t('common.searchPlaceholder')" :aria-label="t('admin.accounts.allGroups')" />
        <button type="button" class="mb-1 w-full rounded px-2 py-1 text-left text-sm hover:bg-gray-100 dark:hover:bg-dark-700" @click="updateGroup([])">{{ t('admin.accounts.allGroups') }}</button>
        <div class="max-h-60 overflow-auto">
          <label v-for="option in filteredGroupOptions" :key="option.value" class="flex cursor-pointer items-center gap-2 rounded px-2 py-2 text-sm hover:bg-gray-100 dark:hover:bg-dark-700">
            <input type="checkbox" :checked="selectedGroups.includes(option.value)" @change="toggleGroup(option.value)" />
            <span class="break-words">{{ option.label }}</span>
          </label>
        </div>
      </div>
    </details>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'; import { useI18n } from 'vue-i18n'; import Select from '@/components/common/Select.vue'; import SearchInput from '@/components/common/SearchInput.vue'
import type { AdminGroup } from '@/types'
import { CONCRETE_PLATFORM_OPTIONS } from '@/constants/platforms'
const props = defineProps<{ searchQuery: string; filters: Record<string, any>; groups?: AdminGroup[] }>()
const emit = defineEmits(['update:searchQuery', 'update:filters', 'change']); const { t } = useI18n()
const updatePlatform = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, platform: value }) }
const updateType = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, type: value }) }
const updateStatus = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, status: value }) }
const updatePrivacyMode = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, privacy_mode: value }) }
const groupSearch = ref('')
const selectedGroups = computed<string[]>(() => String(props.filters.group || '').split(',').filter(Boolean))
const selectedGroupLabels = computed(() => selectedGroups.value.map(id => gOpts.value.find(option => option.value === id)?.label || `#${id}`).join('、'))
const filteredGroupOptions = computed(() => gOpts.value.filter(option => option.value && option.label.toLowerCase().includes(groupSearch.value.toLowerCase())))
const updateGroup = (values: string[]) => { emit('update:filters', { ...props.filters, group: values.join(',') }); emit('change') }
const toggleGroup = (value: string) => updateGroup(selectedGroups.value.includes(value) ? selectedGroups.value.filter(id => id !== value) : [...selectedGroups.value, value])
const pOpts = computed(() => [{ value: '', label: t('admin.accounts.allPlatforms') }, ...CONCRETE_PLATFORM_OPTIONS])
const tOpts = computed(() => [{ value: '', label: t('admin.accounts.allTypes') }, { value: 'oauth', label: t('admin.accounts.oauthType') }, { value: 'setup-token', label: t('admin.accounts.setupToken') }, { value: 'apikey', label: t('admin.accounts.apiKey') }, { value: 'bedrock', label: 'AWS Bedrock' }])
const sOpts = computed(() => [{ value: '', label: t('admin.accounts.allStatus') }, { value: 'active', label: t('admin.accounts.status.active') }, { value: 'inactive', label: t('admin.accounts.status.inactive') }, { value: 'error', label: t('admin.accounts.status.error') }, { value: 'rate_limited', label: t('admin.accounts.status.rateLimited') }, { value: 'temp_unschedulable', label: t('admin.accounts.status.tempUnschedulable') }, { value: 'unschedulable', label: t('admin.accounts.status.unschedulable') }])
const privacyOpts = computed(() => [
  { value: '', label: t('admin.accounts.allPrivacyModes') },
  { value: '__unset__', label: t('admin.accounts.privacyUnset') },
  { value: 'training_off', label: 'Privacy' },
  { value: 'training_set_cf_blocked', label: 'CF' },
  { value: 'training_set_failed', label: 'Fail' }
])
const gOpts = computed(() => [
  { value: '', label: t('admin.accounts.allGroups') },
  { value: 'ungrouped', label: t('admin.accounts.ungroupedGroup') },
  ...(props.groups || []).map(g => ({ value: String(g.id), label: g.name }))
])
</script>
