<template>
  <div :class="flat ? 'p-4 sm:p-6' : 'card p-6'">
    <!-- Toolbar: left filters (multi-line) + right actions -->
    <div class="flex flex-wrap items-end justify-between gap-4">
      <!-- Left: filters (allowed to wrap to multiple rows) -->
      <div class="flex flex-1 flex-wrap items-end gap-4">
        <div class="w-full sm:w-auto sm:min-w-[240px]">
          <label class="input-label">{{ t('admin.usage.userFilter') }}</label>
          <Select :model-value="filters.user_id" :options="userOptions"
            searchable remote clearable :loading="userLoading"
            @search="searchUser" @change="changeUser" />
        </div>

        <div class="w-full sm:w-auto sm:min-w-[240px]">
          <label class="input-label">{{ t('usage.apiKeyFilter') }}</label>
          <Select :model-value="filters.api_key_id" :options="apiKeyOptions"
            searchable remote clearable :loading="apiKeyLoading"
            @search="searchApiKey" @change="changeApiKey" />
        </div>

        <!-- Model Filter -->
        <div class="w-full sm:w-auto sm:min-w-[220px]">
          <label class="input-label">{{ t('usage.model') }}</label>
          <Select v-model="filters.model" :options="modelOptions" searchable @change="emitChange" />
        </div>

        <div class="w-full sm:w-auto sm:min-w-[240px]">
          <label class="input-label">{{ t('admin.usage.account') }}</label>
          <Select :model-value="filters.account_id" :options="accountOptions"
            searchable remote clearable :loading="accountLoading"
            @search="searchAccount" @change="changeAccount" />
        </div>

        <!-- Request Type Filter (usage only) -->
        <div v-if="mode !== 'errors'" class="w-full sm:w-auto sm:min-w-[180px]">
          <label class="input-label">{{ t('usage.type') }}</label>
          <Select v-model="filters.request_type" :options="requestTypeOptions" @change="emitChange" />
        </div>

        <!-- Native compaction is independent of the transport request type. -->
        <div v-if="mode !== 'errors'" class="w-full sm:w-auto sm:min-w-[180px]">
          <label class="input-label">{{ t('usage.compactionFilter') }}</label>
          <Select v-model="filters.native_compaction_v2" :options="compactionOptions" @change="emitChange" />
        </div>

        <!-- Billing Type Filter (usage only) -->
        <div v-if="mode !== 'errors'" class="w-full sm:w-auto sm:min-w-[200px]">
          <label class="input-label">{{ t('admin.usage.billingType') }}</label>
          <Select v-model="filters.billing_type" :options="billingTypeOptions" @change="emitChange" />
        </div>

        <!-- Billing Mode Filter (usage only；用户排行的 user-breakdown 接口不支持该维度) -->
        <div v-if="mode === 'usage'" class="w-full sm:w-auto sm:min-w-[200px]">
          <label class="input-label">{{ t('admin.usage.billingMode') }}</label>
          <Select v-model="filters.billing_mode" :options="billingModeOptions" @change="emitChange" />
        </div>

        <div v-if="mode === 'usage'" class="w-full sm:w-auto sm:min-w-[220px]">
          <label class="input-label">{{ t('admin.usage.upstreamModelAudit') }}</label>
          <Select v-model="filters.upstream_model_mismatch" :options="upstreamModelMismatchOptions" @change="emitChange" />
        </div>

        <div v-if="mode === 'usage'" class="w-full sm:w-auto sm:w-48">
          <label for="min-first-token" class="input-label">{{ t('admin.usage.minFirstToken') }}</label>
          <input id="min-first-token" class="input w-full" type="number" min="0" max="2147483.647" step="0.001"
            list="min-first-token-options" :placeholder="t('admin.usage.minFirstTokenPlaceholder')"
            :value="filters.min_first_token_ms == null ? '' : filters.min_first_token_ms / 1000"
            @change="changeMinFirstToken" />
          <datalist id="min-first-token-options">
            <option v-for="seconds in [1, 2, 5, 10, 30, 60]" :key="seconds" :value="seconds" />
          </datalist>
        </div>

        <!-- Error Phase Filter (errors only) -->
        <div v-if="mode === 'errors'" class="w-full sm:w-auto sm:min-w-[180px]">
          <label class="input-label">{{ t('admin.ops.errorLog.type') }}</label>
          <Select v-model="filters.error_phase" :options="errorPhaseOptions" @change="emitChange" />
        </div>

        <!-- Error Category Filter (errors only) -->
        <div v-if="mode === 'errors'" class="w-full sm:w-auto sm:min-w-[180px]">
          <label class="input-label">{{ t('usage.errors.category') }}</label>
          <Select v-model="filters.error_category" :options="errorCategoryOptions" @change="emitChange" />
        </div>

        <!-- Status Code Filter (errors only) -->
        <div v-if="mode === 'errors'" class="w-full sm:w-auto sm:min-w-[180px]">
          <label class="input-label">{{ t('admin.ops.errorLog.status') }}</label>
          <Select v-model="filters.status_code" :options="statusCodeOptions" @change="emitChange" />
        </div>

        <!-- Group Filter -->
        <div class="w-full sm:w-auto sm:min-w-[200px]">
          <label class="input-label">{{ t('admin.usage.group') }}</label>
          <Select v-model="filters.group_id" :options="groupOptions" searchable @change="emitChange" />
        </div>

      </div>

      <!-- Right: actions -->
      <div v-if="showActions" class="flex w-full flex-wrap items-center justify-end gap-3 sm:w-auto">
        <button type="button" @click="$emit('refresh')" class="btn btn-secondary">
          {{ t('common.refresh') }}
        </button>
        <button type="button" @click="$emit('reset')" class="btn btn-secondary">
          {{ t('common.reset') }}
        </button>
        <slot name="after-reset" />
        <template v-if="mode === 'usage'">
          <button type="button" :disabled="filters.min_first_token_ms != null" :title="filters.min_first_token_ms != null ? t('admin.usage.minFirstTokenCleanup') : undefined" @click="$emit('cleanup')" class="btn btn-danger">
            {{ t('admin.usage.cleanup.button') }}
          </button>
          <button type="button" @click="$emit('export')" :disabled="exporting" class="btn btn-primary">
            {{ t('usage.exportExcel') }}
          </button>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, toRef, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePersistedFilters } from '@/composables/usePersistedFilters'
import { adminAPI } from '@/api/admin'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import { COMMON_ERROR_STATUS_CODES } from '@/utils/errorBadges'
import type { SimpleApiKey, SimpleUser } from '@/api/admin/usage'

type ModelValue = Record<string, any>

interface Props {
  persistenceKey?: string
  modelValue: ModelValue
  exporting: boolean
  startDate: string
  endDate: string
  showActions?: boolean
  modelOptions?: string[]
  /**
   * errors 模式:隐藏用量专属字段/按钮,显示错误类型+状态码(错误请求 tab 用)
   * ranking 模式:同 usage 但隐藏计费模式筛选与清理/导出按钮(用户排行 tab 用)
   */
  mode?: 'usage' | 'errors' | 'ranking'
  /** 嵌入统一卡片内使用：去掉自身卡片外观 */
  flat?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  showActions: true,
  mode: 'usage',
  flat: false
})
const emit = defineEmits([
  'update:modelValue',
  'change',
  'refresh',
  'reset',
  'export',
  'cleanup'
])

const { t } = useI18n()
const changeMinFirstToken = (event: Event) => {
  const input = event.target as HTMLInputElement
  if (!input.validity.valid) { input.reportValidity(); return }
  const value = input.value.trim()
  filters.value.min_first_token_ms = value === '' ? undefined : Math.round(Number(value) * 1000)
  emitChange()
}

const filters = toRef(props, 'modelValue')

const userKeyword = ref('')
const apiKeyKeyword = ref('')
const accountKeyword = ref('')
const userResults = ref<SimpleUser[]>([])
const apiKeyResults = ref<SimpleApiKey[]>([])
const accountResults = ref<{ id: number; name: string }[]>([])
const userLoading = ref(false)
const apiKeyLoading = ref(false)
const accountLoading = ref(false)
let userSearchSequence = 0
let apiKeySearchSequence = 0
let accountSearchSequence = 0

function optionsWithSelection(options: SelectOption[], id: number | undefined, label: string): SelectOption[] {
  if (id && !options.some(option => option.value === id)) {
    options.unshift({ value: id, label: label || `#${id}` })
  }
  return [{ value: null, label: t('common.all') }, ...options]
}
const userOptions = computed(() => optionsWithSelection(userResults.value.map(u => ({
  value: u.id, label: `${u.email}${u.deleted ? ` (${t('admin.usage.userDeletedBadge')})` : ''} #${u.id}`
})), filters.value.user_id, userKeyword.value))
const apiKeyOptions = computed(() => optionsWithSelection(apiKeyResults.value.map(k => ({
  value: k.id, label: `${k.name || k.id} #${k.id}`
})), filters.value.api_key_id, apiKeyKeyword.value))
const accountOptions = computed(() => optionsWithSelection(accountResults.value.map(a => ({
  value: a.id, label: `${a.name} #${a.id}`
})), filters.value.account_id, accountKeyword.value))

const modelOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('admin.usage.allModels') },
  ...(props.modelOptions ?? []).map((m) => ({ value: m, label: m })),
])
const groupOptions = ref<SelectOption[]>([{ value: null, label: t('admin.usage.allGroups') }])

const requestTypeOptions = ref<SelectOption[]>([
  { value: null, label: t('admin.usage.allTypes') },
  { value: 'ws_v2', label: t('usage.ws') },
  { value: 'live', label: t('usage.live') },
  { value: 'stream', label: t('usage.stream') },
  { value: 'sync', label: t('usage.sync') },
  { value: 'cyber', label: t('usage.cyber') }
])

const compactionOptions = ref<SelectOption[]>([
  { value: null, label: t('usage.allCompactionTypes') },
  { value: true, label: t('usage.compactionOnly') }
])

const billingTypeOptions = ref<SelectOption[]>([
  { value: null, label: t('admin.usage.allBillingTypes') },
  { value: 0, label: t('admin.usage.billingTypeBalance') },
  { value: 1, label: t('admin.usage.billingTypeSubscription') }
])

// 错误类型对应后端 phase 参数(与错误表"类型"徽章同语义)
const errorPhaseOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('admin.usage.allTypes') },
  { value: 'upstream', label: t('admin.ops.errorLog.typeUpstream') },
  { value: 'account_auth', label: t('admin.ops.errorLog.typeAccountAuth') },
  { value: 'request', label: t('admin.ops.errorLog.typeRequest') },
  { value: 'auth', label: t('admin.ops.errorLog.typeAuth') },
  { value: 'routing', label: t('admin.ops.errorLog.typeRouting') },
  { value: 'internal', label: t('admin.ops.errorLog.typeInternal') },
])

// 分类码同用户端 /usage 错误筛选;"other" 无法反查为过滤条件,刻意不列
const errorCategoryCodes = ['auth', 'rate_limit', 'quota', 'invalid_request', 'service_unavailable', 'upstream', 'internal', 'cyber']

const errorCategoryOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('usage.errors.allCategories') },
  ...errorCategoryCodes.map((c) => ({ value: c, label: t('usage.errors.categories.' + c) })),
])

const statusCodeOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('usage.errors.allStatuses') },
  ...COMMON_ERROR_STATUS_CODES.map((c) => ({ value: c, label: String(c) })),
])

const billingModeOptions = ref<SelectOption[]>([
  { value: null, label: t('admin.usage.allBillingModes') },
  { value: 'token', label: t('admin.usage.billingModeToken') },
  { value: 'per_request', label: t('admin.usage.billingModePerRequest') },
  { value: 'image', label: t('admin.usage.billingModeImage') },
  { value: 'video', label: t('admin.usage.billingModeVideo') }
])

const upstreamModelMismatchOptions = ref<SelectOption[]>([
  { value: null, label: t('admin.usage.allUpstreamModelAudit') },
  { value: true, label: t('admin.usage.upstreamModelMismatchOnly') },
  { value: false, label: t('admin.usage.upstreamModelMatchedOnly') }
])

const emitChange = () => emit('change')

const searchUser = async (query = '') => {
  const seq = ++userSearchSequence
  userLoading.value = true
  try {
    const results = await adminAPI.usage.searchUsers(query)
    if (seq === userSearchSequence) userResults.value = results.sort((a, b) => Number(a.deleted) - Number(b.deleted))
  } catch {
    if (seq === userSearchSequence) userResults.value = []
  } finally {
    if (seq === userSearchSequence) userLoading.value = false
  }
}
const searchApiKey = async (query = '') => {
  const seq = ++apiKeySearchSequence
  apiKeyLoading.value = true
  try {
    const results = await adminAPI.usage.searchApiKeys(filters.value.user_id, query)
    if (seq === apiKeySearchSequence) apiKeyResults.value = results
  } catch {
    if (seq === apiKeySearchSequence) apiKeyResults.value = []
  } finally {
    if (seq === apiKeySearchSequence) apiKeyLoading.value = false
  }
}
const searchAccount = async (query = '') => {
  const seq = ++accountSearchSequence
  accountLoading.value = true
  try {
    const results = await adminAPI.accounts.list(1, query ? 30 : 1000, { search: query })
    if (seq === accountSearchSequence) accountResults.value = results.items.map(a => ({ id: a.id, name: a.name }))
  } catch {
    if (seq === accountSearchSequence) accountResults.value = []
  } finally {
    if (seq === accountSearchSequence) accountLoading.value = false
  }
}
const changeUser = (value: string | number | boolean | null, option: SelectOption | null) => {
  const id = typeof value === 'number' ? value : undefined
  filters.value.user_id = id || undefined
  userKeyword.value = id ? String(option?.label || '') : ''
  filters.value.api_key_id = undefined
  apiKeyKeyword.value = ''
  apiKeyResults.value = []
  emitChange()
}
const changeApiKey = (value: string | number | boolean | null, option: SelectOption | null) => {
  const id = typeof value === 'number' ? value : undefined
  filters.value.api_key_id = id || undefined
  apiKeyKeyword.value = id ? String(option?.label || '') : ''
  emitChange()
}
const changeAccount = (value: string | number | boolean | null, option: SelectOption | null) => {
  const id = typeof value === 'number' ? value : undefined
  filters.value.account_id = id || undefined
  accountKeyword.value = id ? String(option?.label || '') : ''
  emitChange()
}

watch(
  () => props.startDate,
  (value) => {
    filters.value.start_date = value
  },
  { immediate: true }
)

watch(
  () => props.endDate,
  (value) => {
    filters.value.end_date = value
  },
  { immediate: true }
)

watch(
  () => filters.value.user_id,
  (userId) => {
    void searchApiKey()
    if (!userId) {
      userKeyword.value = ''
    }
  }
)

watch(
  () => filters.value.api_key_id,
  (apiKeyId) => {
    if (!apiKeyId) {
      apiKeyKeyword.value = ''
    }
  }
)

watch(
  () => filters.value.account_id,
  (accountId) => {
    if (!accountId) {
      accountKeyword.value = ''
    }
  }
)

onMounted(async () => {
  void searchUser()
  void searchApiKey()
  void searchAccount()
  try {
    const gs = await adminAPI.groups.list(1, 1000)
    groupOptions.value.push(...gs.items.map((g: any) => ({ value: g.id, label: g.name })))
  } catch {
    // Ignore filter option loading errors (page still usable)
  }
})

onUnmounted(() => {
  userSearchSequence++
  apiKeySearchSequence++
  accountSearchSequence++
})

// 供外部(如用户排行下钻)在程序化设置 user_id 后回显选中的用户邮箱
const setUserKeyword = (email: string) => {
  userSearchSequence++
  userKeyword.value = email
}

const getUserSearchRevision = () => userSearchSequence

defineExpose({ getUserSearchRevision, setUserKeyword })

if (props.persistenceKey) {
  usePersistedFilters(props.persistenceKey, { userKeyword, apiKeyKeyword, accountKeyword })
}
</script>
