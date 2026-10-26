<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Loading State -->
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>

      <template v-else-if="stats">
        <!-- Responsive stats: performance first, full width on mobile -->
        <div class="grid grid-cols-2 gap-3 sm:gap-4 lg:grid-cols-4">
          <!-- Performance (RPM/TPM) -->
          <div class="card col-span-2 min-w-0 p-3 sm:col-span-1 sm:p-4">
            <div class="flex flex-wrap items-center gap-2 sm:gap-3">
              <div class="shrink-0 rounded-lg bg-violet-100 p-2 dark:bg-violet-900/30">
                <Icon name="bolt" size="md" class="text-violet-600 dark:text-violet-400" :stroke-width="2" />
              </div>
              <div class="min-w-max flex-1">
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.performance') }}
                </p>
                <div class="flex items-baseline gap-2">
                  <p class="text-xl font-bold text-gray-900 dark:text-white">
                    {{ performance ? formatTokens(performance.rpm) : '—' }}
                  </p>
                  <span class="text-xs text-gray-500 dark:text-gray-400">RPM</span>
                </div>
                <div class="flex items-baseline gap-2">
                  <p class="text-sm font-semibold text-violet-600 dark:text-violet-400">
                    {{ performance ? formatTokens(performance.tpm) : '—' }}
                  </p>
                  <span class="text-xs text-gray-500 dark:text-gray-400">TPM</span>
                </div>
              </div>
              <div class="ml-auto shrink-0 border-l border-gray-100 pl-2 text-right sm:pl-3 dark:border-dark-700" data-testid="recent-errors" :title="t('admin.dashboard.recentErrorsHint')">
                <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.recentErrors') }}</p>
                <p class="text-xl font-bold tabular-nums" :class="performance?.error_count ? 'text-red-600 dark:text-red-400' : 'text-gray-900 dark:text-white'">
                  {{ performance?.error_count != null ? formatNumber(performance.error_count) : '—' }}
                </p>
                <div class="mt-1" data-testid="hour-errors" :title="t('admin.dashboard.hourErrorsHint')">
                  <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.hourErrors') }}</p>
                  <p class="text-sm font-semibold tabular-nums" :class="performance?.hour_error_count ? 'text-red-600 dark:text-red-400' : 'text-gray-900 dark:text-white'">
                    {{ performance?.hour_error_count != null ? formatNumber(performance.hour_error_count) : '—' }}
                  </p>
                </div>
              </div>
            </div>
            <p class="mt-2 flex flex-wrap items-baseline gap-x-2 text-xs text-gray-500 dark:text-gray-400" data-testid="error-free-time" :title="t('admin.dashboard.errorFreeTimeHint')">
              <span>{{ t('admin.dashboard.errorFreeTime') }}</span>
              <strong class="text-emerald-600 dark:text-emerald-400">{{ errorFreeTime }}</strong>
            </p>
            <p class="mt-2 text-xs text-gray-600 dark:text-gray-300" data-testid="recent-ttft">
              {{ t('admin.dashboard.recentFirstToken') }}:
              <strong>{{ formatFirstToken(performance?.avg_first_token_ms) }}</strong>
              <span v-if="performance"> · {{ t('admin.dashboard.firstTokenSamples', { count: performance.first_token_samples ?? 0 }) }}</span>
            </p>
            <div v-if="performance?.kiro_rs_accounts?.length" class="mt-2 border-t border-gray-100 pt-2 text-xs dark:border-dark-700" data-testid="kiro-rs-accounts">
              <p class="mb-1 text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.kiroPriorityZero') }}</p>
              <div v-for="instance in performance.kiro_rs_accounts" :key="instance.name" class="flex items-baseline justify-between gap-2 py-0.5" :title="instance.updated_at ? new Date(instance.updated_at).toLocaleString() : ''">
                <span class="min-w-0 break-all text-gray-600 dark:text-gray-300">{{ instance.name }}</span>
                <strong class="shrink-0 tabular-nums" :class="instance.status === 'ok' ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400'">{{ instance.status === 'ok' && instance.priority_zero_available != null ? formatNumber(instance.priority_zero_available) : t(instance.status === 'loading' ? 'admin.dashboard.kiroLoading' : 'admin.dashboard.kiroUnavailable') }}</strong>
              </div>
            </div>
            <p v-if="performanceError" role="status" class="mt-1 text-xs text-amber-600 dark:text-amber-400">
              {{ t('admin.dashboard.performanceRefreshFailed') }}
            </p>
            <div v-if="performance" class="mt-2 max-h-48 overflow-auto">
              <table class="w-full table-fixed text-xs" data-testid="model-performance">
                <tbody>
                  <template v-for="item in performance.models" :key="item.model">
                    <tr>
                      <th colspan="3" scope="rowgroup" class="break-all pt-1.5 pb-0.5 text-left font-medium text-gray-700 dark:text-gray-300" :title="item.model">{{ item.model }}</th>
                    </tr>
                    <tr class="border-b border-gray-100 dark:border-dark-700">
                      <td colspan="3" class="pb-1.5 tabular-nums">
                        <div class="flex flex-wrap items-baseline gap-x-3 gap-y-0.5" data-testid="model-performance-values">
                          <span class="inline-flex items-baseline gap-1 whitespace-nowrap">
                            <span class="text-gray-500 dark:text-gray-400">RPM</span>
                            <span>{{ formatTokens(item.rpm) }}</span>
                          </span>
                          <span class="inline-flex items-baseline gap-1 whitespace-nowrap">
                            <span class="text-gray-500 dark:text-gray-400">TPM</span>
                            <span>{{ formatTokens(item.tpm) }}</span>
                          </span>
                          <span class="inline-flex items-baseline gap-1 whitespace-nowrap" :title="t('admin.dashboard.firstTokenSamples', { count: item.first_token_samples ?? 0 })">
                            <span class="text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.firstTokenAverage') }}</span>
                            <span>{{ formatFirstToken(item.avg_first_token_ms) }}</span>
                          </span>
                        </div>
                      </td>
                    </tr>
                  </template>
                  <tr v-if="performance.models.length === 0">
                    <td colspan="3" class="py-2 text-gray-500">{{ t('admin.dashboard.noDataAvailable') }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p v-if="performance" class="mt-2 text-xs text-gray-400">
              {{ t('admin.dashboard.performanceUpdatedAt', { time: new Date(performance.updated_at).toLocaleTimeString() }) }}
            </p>
          </div>

          <!-- Today Requests -->
          <div class="card col-span-2 min-w-0 p-3 sm:col-span-1 sm:p-4">
            <div class="flex flex-col items-start gap-2 sm:flex-row sm:items-center sm:gap-3">
              <div class="rounded-lg bg-green-100 p-2 dark:bg-green-900/30">
                <Icon name="chart" size="md" class="text-green-600 dark:text-green-400" :stroke-width="2" />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.todayRequests') }}
                </p>
                <p class="text-xl font-bold text-gray-900 dark:text-white">
                  {{ formatNumber(summaryTotals.todayRequests) }}
                </p>
                <p class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('common.total') }}: {{ formatNumber(stats.total_requests) }}
                </p>
              </div>
              <div class="min-w-0 sm:ml-auto sm:border-l sm:border-gray-100 sm:pl-3 dark:sm:border-dark-700" data-testid="today-errors" :title="t('admin.dashboard.todayErrorsHint')">
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.dashboard.todayErrors') }}</p>
                <div class="flex flex-wrap items-baseline gap-x-2">
                  <span class="text-xl font-bold text-red-600 dark:text-red-400">{{ performance?.today_error_count == null ? '—' : formatNumber(performance.today_error_count) }}</span>
                  <span class="text-sm tabular-nums text-red-600 dark:text-red-400">{{ performance?.today_error_rate == null ? '—' : `${performance.today_error_rate.toFixed(2)}%` }}</span>
                </div>
                <p v-if="performanceError" role="status" class="text-xs text-amber-600">{{ t('admin.dashboard.failedToLoad') }}</p>
              </div>
            </div>
            <div class="mt-3 max-h-48 overflow-auto" data-testid="today-requests-models">
              <table v-if="performance?.today_models || modelSummary" class="w-full table-fixed text-xs">
                <thead><tr class="text-gray-500 dark:text-gray-400">
                  <th class="w-1/2 py-1 text-left font-medium">{{ t('usage.model') }}</th>
                  <th class="py-1 text-right font-medium">{{ t('admin.dashboard.todayRequests') }}</th>
                  <th class="py-1 text-right font-medium">{{ t('admin.dashboard.errorsAndRate') }}</th>
                </tr></thead>
                <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                  <tr v-for="item in todayRequestModels" :key="item.model">
                    <td class="break-words py-1.5 pr-2">{{ item.model }}</td>
                    <td class="py-1.5 text-right tabular-nums" :title="formatNumber(item.requests)">{{ formatNumber(item.requests) }}</td>
                    <td class="py-1.5 pl-2 text-right tabular-nums" :class="item.error_count ? 'text-red-600 dark:text-red-400' : 'text-gray-500'">
                      <span class="block">{{ item.error_count == null ? '—' : formatNumber(item.error_count) }}</span>
                      <span class="block text-[11px]">{{ item.error_rate == null ? '—' : `${item.error_rate.toFixed(2)}%` }}</span>
                    </td>
                  </tr>
                  <tr v-if="!todayRequestModels.length"><td colspan="3" class="py-2 text-gray-400">{{ t('common.noData') }}</td></tr>
                </tbody>
              </table>
              <p v-else class="text-xs text-gray-400">{{ t('common.loading') }}</p>
              <p v-if="modelSummaryError" role="status" class="mt-1 text-xs text-amber-600">{{ t('admin.dashboard.failedToLoad') }}</p>
            </div>
          </div>

          <!-- Today Tokens -->
          <div class="card col-span-2 min-w-0 p-3 sm:col-span-1 sm:p-4">
            <div class="flex flex-col items-start gap-2 sm:flex-row sm:items-center sm:gap-3">
              <div class="rounded-lg bg-amber-100 p-2 dark:bg-amber-900/30">
                <Icon name="cube" size="md" class="text-amber-600 dark:text-amber-400" :stroke-width="2" />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.todayTokens') }}
                </p>
                <p class="text-xl font-bold text-gray-900 dark:text-white">
                  {{ formatTokens(summaryTotals.todayTokens) }}
                </p>
                <p class="text-xs">
                  <span
                    class="text-green-600 dark:text-green-400"
                    :title="t('admin.dashboard.actual')"
                    >${{ formatCost(summaryCosts.today.actual_cost) }}</span
                  >
                  <span class="text-gray-400 dark:text-gray-500"> / </span>
                  <span
                    class="text-orange-500 dark:text-orange-400"
                    :title="t('admin.dashboard.accountCost')"
                    >${{ formatCost(summaryCosts.today.account_cost) }}</span
                  >
                  <span class="text-gray-400 dark:text-gray-500"> / </span>
                  <span
                    class="text-gray-400 dark:text-gray-500"
                    :title="t('admin.dashboard.standard')"
                    >${{ formatCost(summaryCosts.today.cost) }}</span
                  >
                </p>
              </div>
            </div>
            <div class="mt-3 max-h-48 overflow-auto" data-testid="today-total_tokens-models">
              <table v-if="modelSummary" class="w-full table-fixed text-xs">
                <thead><tr class="text-gray-500 dark:text-gray-400">
                  <th class="w-2/3 py-1 text-left font-medium">{{ t('usage.model') }}</th>
                  <th class="py-1 text-right font-medium">Token</th>
                </tr></thead>
                <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                  <template v-for="item in modelSummary.today" :key="item.model">
                    <tr>
                      <td class="break-words py-1.5 pr-2">{{ item.model }}</td>
                      <td class="py-1.5 text-right tabular-nums" :title="formatNumber(item.total_tokens)">{{ formatTokens(item.total_tokens) }}</td>
                    </tr>
                    <tr>
                      <td colspan="2" class="pb-2">
                        <dl class="flex flex-wrap gap-x-3 gap-y-1 text-[11px]" data-testid="today-model-costs">
                          <div class="text-green-600 dark:text-green-400"><dt class="inline">{{ t('admin.dashboard.actual') }}：</dt><dd class="inline tabular-nums">${{ formatCost(item.actual_cost) }}</dd></div>
                          <div class="text-orange-500 dark:text-orange-400"><dt class="inline">{{ t('admin.dashboard.accountCost') }}：</dt><dd class="inline tabular-nums">${{ formatCost(item.account_cost) }}</dd></div>
                          <div class="text-gray-500 dark:text-gray-400"><dt class="inline">{{ t('admin.dashboard.standard') }}：</dt><dd class="inline tabular-nums">${{ formatCost(item.cost) }}</dd></div>
                        </dl>
                      </td>
                    </tr>
                  </template>
                  <tr v-if="!modelSummary.today.length"><td colspan="2" class="py-2 text-gray-400">{{ t('common.noData') }}</td></tr>
                </tbody>
              </table>
              <p v-else class="text-xs text-gray-400">{{ t('common.loading') }}</p>
              <p v-if="modelSummaryError" role="status" class="mt-1 text-xs text-amber-600">{{ t('admin.dashboard.failedToLoad') }}</p>
            </div>
          </div>

          <!-- Total Tokens -->
          <div class="card col-span-2 min-w-0 p-3 sm:col-span-1 sm:p-4">
            <div class="flex flex-col items-start gap-2 sm:flex-row sm:items-center sm:gap-3">
              <div class="rounded-lg bg-indigo-100 p-2 dark:bg-indigo-900/30">
                <Icon name="database" size="md" class="text-indigo-600 dark:text-indigo-400" :stroke-width="2" />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.totalTokens') }}
                </p>
                <p class="text-xl font-bold text-gray-900 dark:text-white">
                  {{ formatTokens(summaryTotals.totalTokens) }}
                </p>
                <p class="text-xs">
                  <span
                    class="text-green-600 dark:text-green-400"
                    :title="t('admin.dashboard.actual')"
                    >${{ formatCost(summaryCosts.total.actual_cost) }}</span
                  >
                  <span class="text-gray-400 dark:text-gray-500"> / </span>
                  <span
                    class="text-orange-500 dark:text-orange-400"
                    :title="t('admin.dashboard.accountCost')"
                    >${{ formatCost(summaryCosts.total.account_cost) }}</span
                  >
                  <span class="text-gray-400 dark:text-gray-500"> / </span>
                  <span
                    class="text-gray-400 dark:text-gray-500"
                    :title="t('admin.dashboard.standard')"
                    >${{ formatCost(summaryCosts.total.cost) }}</span
                  >
                </p>
              </div>
            </div>
            <div class="mt-3 max-h-48 overflow-auto" data-testid="total-total_tokens-models">
              <table v-if="modelSummary" class="w-full table-fixed text-xs">
                <thead><tr class="text-gray-500 dark:text-gray-400">
                  <th class="w-2/3 py-1 text-left font-medium">{{ t('usage.model') }}</th>
                  <th class="py-1 text-right font-medium">Token</th>
                </tr></thead>
                <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                  <template v-for="item in modelSummary.total" :key="item.model">
                    <tr>
                      <td class="break-words py-1.5 pr-2">{{ item.model }}</td>
                      <td class="py-1.5 text-right tabular-nums" :title="formatNumber(item.total_tokens)">{{ formatTokens(item.total_tokens) }}</td>
                    </tr>
                    <tr>
                      <td colspan="2" class="pb-2">
                        <dl class="flex flex-wrap gap-x-3 gap-y-1 text-[11px]" data-testid="total-model-costs">
                          <div class="text-green-600 dark:text-green-400"><dt class="inline">{{ t('admin.dashboard.actual') }}：</dt><dd class="inline tabular-nums">${{ formatCost(item.actual_cost) }}</dd></div>
                          <div class="text-orange-500 dark:text-orange-400"><dt class="inline">{{ t('admin.dashboard.accountCost') }}：</dt><dd class="inline tabular-nums">${{ formatCost(item.account_cost) }}</dd></div>
                          <div class="text-gray-500 dark:text-gray-400"><dt class="inline">{{ t('admin.dashboard.standard') }}：</dt><dd class="inline tabular-nums">${{ formatCost(item.cost) }}</dd></div>
                        </dl>
                      </td>
                    </tr>
                  </template>
                  <tr v-if="!modelSummary.total.length"><td colspan="2" class="py-2 text-gray-400">{{ t('common.noData') }}</td></tr>
                </tbody>
              </table>
              <p v-else class="text-xs text-gray-400">{{ t('common.loading') }}</p>
              <p v-if="modelSummaryError" role="status" class="mt-1 text-xs text-amber-600">{{ t('admin.dashboard.failedToLoad') }}</p>
            </div>
          </div>

          <!-- Avg Response Time -->
          <div class="card min-w-0 p-3 sm:p-4">
            <div class="flex flex-col items-start gap-2 sm:flex-row sm:items-center sm:gap-3">
              <div class="rounded-lg bg-rose-100 p-2 dark:bg-rose-900/30">
                <Icon name="clock" size="md" class="text-rose-600 dark:text-rose-400" :stroke-width="2" />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.avgResponse') }}
                </p>
                <p class="text-xl font-bold text-gray-900 dark:text-white">
                  {{ formatDuration(stats.average_duration_ms) }}
                </p>
                <p class="text-xs text-gray-500 dark:text-gray-400">
                  {{ stats.active_users }} {{ t('admin.dashboard.activeUsers') }}
                </p>
              </div>
            </div>
          </div>

          <!-- New Users Today -->
          <div class="card min-w-0 p-3 sm:p-4">
            <div class="flex flex-col items-start gap-2 sm:flex-row sm:items-center sm:gap-3">
              <div class="rounded-lg bg-emerald-100 p-2 dark:bg-emerald-900/30">
                <Icon name="userPlus" size="md" class="text-emerald-600 dark:text-emerald-400" :stroke-width="2" />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.users') }}
                </p>
                <p class="text-xl font-bold text-emerald-600 dark:text-emerald-400">
                  +{{ stats.today_new_users }}
                </p>
                <p class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('common.total') }}: {{ formatNumber(stats.total_users) }}
                </p>
              </div>
            </div>
          </div>

          <!-- Total API Keys -->
          <div class="card min-w-0 p-3 sm:p-4">
            <div class="flex flex-col items-start gap-2 sm:flex-row sm:items-center sm:gap-3">
              <div class="rounded-lg bg-blue-100 p-2 dark:bg-blue-900/30">
                <Icon name="key" size="md" class="text-blue-600 dark:text-blue-400" :stroke-width="2" />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.apiKeys') }}
                </p>
                <p class="text-xl font-bold text-gray-900 dark:text-white">
                  {{ stats.total_api_keys }}
                </p>
                <p class="text-xs text-green-600 dark:text-green-400">
                  {{ stats.active_api_keys }} {{ t('common.active') }}
                </p>
              </div>
            </div>
          </div>

          <!-- Service Accounts -->
          <div class="card col-span-2 min-w-0 p-3 sm:col-span-1 sm:p-4">
            <div class="flex flex-col items-start gap-2 sm:flex-row sm:items-center sm:gap-3">
              <div class="rounded-lg bg-purple-100 p-2 dark:bg-purple-900/30">
                <Icon name="server" size="md" class="text-purple-600 dark:text-purple-400" :stroke-width="2" />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.accounts') }}
                </p>
                <p class="text-xl font-bold text-gray-900 dark:text-white">
                  {{ stats.total_accounts }}
                </p>
                <p class="text-xs">
                  <span class="text-green-600 dark:text-green-400"
                    >{{ stats.normal_accounts }} {{ t('common.active') }}</span
                  >
                  <span v-if="stats.error_accounts > 0" class="ml-1 text-red-500"
                    >{{ stats.error_accounts }} {{ t('common.error') }}</span
                  >
                </p>
              </div>
            </div>
          </div>

        </div>

        <!-- Quick Actions -->
        <div class="card p-4">
          <div class="mb-3 flex items-center justify-between">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('admin.dashboard.quickActions') }}
            </h2>
          </div>
          <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
            <button
              v-if="canUseBatchImage"
              type="button"
              class="group flex items-center gap-3 rounded-lg bg-gray-50 p-3 text-left transition-colors hover:bg-sky-50 dark:bg-dark-800/50 dark:hover:bg-sky-900/20"
              @click="router.push('/batch-image')"
            >
              <span class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-sky-100 text-sky-600 dark:bg-sky-900/30 dark:text-sky-400">
                <Icon name="sparkles" size="md" :stroke-width="2" />
              </span>
              <span class="min-w-0 flex-1">
                <span class="block text-sm font-medium text-gray-900 dark:text-white">
                  {{ t('admin.dashboard.batchImage') }}
                </span>
                <span class="block text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.batchImageDesc') }}
                </span>
              </span>
              <Icon name="chevronRight" size="sm" class="text-gray-400 group-hover:text-sky-500" />
            </button>
            <button
              type="button"
              class="group flex items-center gap-3 rounded-lg bg-gray-50 p-3 text-left transition-colors hover:bg-emerald-50 dark:bg-dark-800/50 dark:hover:bg-emerald-900/20"
              @click="router.push('/admin/groups')"
            >
              <span class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-emerald-100 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-400">
                <Icon name="grid" size="md" :stroke-width="2" />
              </span>
              <span class="min-w-0 flex-1">
                <span class="block text-sm font-medium text-gray-900 dark:text-white">
                  {{ t('admin.dashboard.groupPricing') }}
                </span>
                <span class="block text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.dashboard.groupPricingDesc') }}
                </span>
              </span>
              <Icon name="chevronRight" size="sm" class="text-gray-400 group-hover:text-emerald-500" />
            </button>
          </div>
        </div>

        <!-- Charts Section -->
        <div class="space-y-6">
          <!-- Date Range Filter -->
          <div class="card p-4">
            <div class="flex flex-wrap items-center gap-4">
              <div class="flex items-center gap-2">
                <span class="text-sm font-medium text-gray-700 dark:text-gray-300"
                  >{{ t('admin.dashboard.timeRange') }}:</span
                >
                <DateRangePicker
                  v-model:start-date="startDate"
                  v-model:end-date="endDate"
                  @change="onDateRangeChange"
                />
              </div>
              <button @click="loadDashboardStats" :disabled="chartsLoading" class="btn btn-secondary">
                {{ t('common.refresh') }}
              </button>
              <div class="ml-auto flex items-center gap-2">
                <span class="text-sm font-medium text-gray-700 dark:text-gray-300"
                  >{{ t('admin.dashboard.granularity') }}:</span
                >
                <div class="w-28">
                  <Select
                    v-model="granularity"
                    :options="granularityOptions"
                    @change="loadChartData"
                  />
                </div>
              </div>
            </div>
          </div>

          <!-- Charts Grid -->
          <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
            <ModelDistributionChart
              :model-stats="modelStats"
              :enable-ranking-view="true"
              :ranking-items="rankingItems"
              :ranking-total-actual-cost="rankingTotalActualCost"
              :ranking-total-requests="rankingTotalRequests"
              :ranking-total-tokens="rankingTotalTokens"
              :loading="chartsLoading"
              :ranking-loading="rankingLoading"
              :ranking-error="rankingError"
              :start-date="startDate"
              :end-date="endDate"
              @ranking-click="goToUserUsage"
            />
            <TokenUsageTrend
              :trend-data="trendData"
              :loading="chartsLoading"
              :models="modelStats.map(item => item.model)"
              :model="trendModel"
              @update:model="onTrendModelChange"
            />
          </div>

          <!-- User Usage Trend (Full Width) -->
          <div class="card p-4">
            <div class="mb-4 flex items-center justify-between gap-3">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.dashboard.recentUsage') }} (Top 12)</h3>
              <div class="flex gap-1" role="group" :aria-label="t('admin.dashboard.recentUsage')">
                <button v-for="metric in (['tokens', 'actual_cost'] as const)" :key="metric" type="button"
                  class="rounded px-2 py-1 text-xs" :class="userTrendMetric === metric ? 'bg-blue-600 text-white' : 'text-gray-600 dark:text-gray-300'"
                  :aria-pressed="userTrendMetric === metric" @click="setUserTrendMetric(metric)">
                  {{ t(metric === 'tokens' ? 'admin.dashboard.tokens' : 'admin.dashboard.actualSpending') }}
                </button>
              </div>
            </div>
            <div class="h-64">
              <div v-if="userTrendLoading" class="flex h-full items-center justify-center">
                <LoadingSpinner size="md" />
              </div>
              <Line v-else-if="userTrendChartData" :data="userTrendChartData" :options="lineOptions" />
              <div
                v-else
                class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400"
              >
                {{ t('admin.dashboard.noDataAvailable') }}
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { usePersistedFilters } from '@/composables/usePersistedFilters'

import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
import { adminAPI } from '@/api/admin'
import type { DashboardPerformance } from '@/api/admin/dashboard'
import type {
  DashboardStats,
  TrendDataPoint,
  ModelStat,
  UserUsageTrendPoint,
  UserSpendingRankingItem
} from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import ModelDistributionChart from '@/components/charts/ModelDistributionChart.vue'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'

import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line } from 'vue-chartjs'

// Register Chart.js components
ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
)

const appStore = useAppStore()
const router = useRouter()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()
const stats = ref<DashboardStats | null>(null)
// Independent of the chart date filter; use the same requested-model dimension as performance.
const modelSummary = ref<{ today: ModelStat[]; total: ModelStat[] } | null>(null)
const modelSummaryError = ref(false)
let modelSummaryTimer: ReturnType<typeof setInterval> | undefined
let modelSummaryController: AbortController | null = null
const todayRequestModels = computed(() => performance.value?.today_models ??
  (modelSummary.value?.today ?? []).map(row => ({ model: row.model, requests: row.requests, error_count: undefined, error_rate: undefined })))
const summaryTotals = computed(() => ({
  todayRequests: modelSummary.value?.today.reduce((sum, row) => sum + row.requests, 0) ?? stats.value?.today_requests ?? 0,
  todayTokens: modelSummary.value?.today.reduce((sum, row) => sum + row.total_tokens, 0) ?? stats.value?.today_tokens ?? 0,
  totalTokens: modelSummary.value?.total.reduce((sum, row) => sum + row.total_tokens, 0) ?? stats.value?.total_tokens ?? 0,
}))
const summaryCosts = computed(() => {
  const sumCosts = (rows: ModelStat[] | undefined, period: 'today' | 'total') => ({
    actual_cost: rows ? rows.reduce((sum, row) => sum + (row.actual_cost || 0), 0) : stats.value?.[`${period}_actual_cost`] ?? 0,
    account_cost: rows ? rows.reduce((sum, row) => sum + (row.account_cost || 0), 0) : stats.value?.[`${period}_account_cost`] ?? 0,
    cost: rows ? rows.reduce((sum, row) => sum + (row.cost || 0), 0) : stats.value?.[`${period}_cost`] ?? 0,
  })
  return { today: sumCosts(modelSummary.value?.today, 'today'), total: sumCosts(modelSummary.value?.total, 'total') }
})
const loadModelSummary = async () => {
  if (performanceDisposed || modelSummaryController || document.visibilityState === 'hidden') return
  const controller = new AbortController()
  modelSummaryController = controller
  const now = new Date()
  const today = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
  try {
    const [daily, total] = await Promise.all([
      adminAPI.dashboard.getModelStats({ start_date: today, end_date: today, model_source: 'requested' }, controller.signal),
      adminAPI.dashboard.getModelStats({ start_date: '1970-01-01', end_date: today, model_source: 'requested' }, controller.signal),
    ])
    if (performanceDisposed) return
    modelSummary.value = { today: daily.models || [], total: total.models || [] }
    modelSummaryError.value = false
  } catch {
    if (!performanceDisposed) modelSummaryError.value = true
  } finally {
    modelSummaryController = null
  }
}
const formatFirstToken = (ms: number | null | undefined) => ms == null ? '—' : `${(ms / 1000).toFixed(2)} s`

const performance = ref<DashboardPerformance | null>(null)
const performanceClock = ref(Date.now())
const performanceReceivedAt = ref(performanceClock.value)
const errorFreeTime = computed(() => {
  if (!performance.value) return '—'
  const seconds = performance.value.error_free_seconds
  if (seconds === undefined) return '—'
  if (seconds === null) return t('admin.dashboard.noErrorHistory')
  const elapsed = Math.max(0, performanceClock.value - performanceReceivedAt.value) / 1000
  const total = Math.max(0, Math.floor(seconds + elapsed))
  const days = Math.floor(total / 86400)
  const hours = Math.floor(total % 86400 / 3600)
  const minutes = Math.floor(total % 3600 / 60)
  const clock = `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(total % 60).padStart(2, '0')}`
  return days > 0 ? `${t('admin.dashboard.errorFreeDays', { count: days })} ${clock}` : clock
})
const performanceError = ref(false)
let performanceTimer: ReturnType<typeof setInterval> | undefined
let errorFreeTimer: ReturnType<typeof setInterval> | undefined
let performanceController: AbortController | null = null
let performanceDisposed = false

const loadPerformance = async () => {
  if (performanceDisposed || performanceController || document.visibilityState === 'hidden') return
  const controller = new AbortController()
  performanceController = controller
  try {
    const result = await adminAPI.dashboard.getPerformance(controller.signal)
    if (performanceDisposed) return
    performanceReceivedAt.value = Date.now()
    performanceClock.value = performanceReceivedAt.value
    performance.value = result
    performanceError.value = false
  } catch {
    if (!performanceDisposed) performanceError.value = true
  } finally {
    performanceController = null
  }
}

const loading = ref(false)
const chartsLoading = ref(false)
const userTrendLoading = ref(false)
const rankingLoading = ref(false)
const rankingError = ref(false)

// Chart data
const trendData = ref<TrendDataPoint[]>([])
const trendModel = ref('')
const modelStats = ref<ModelStat[]>([])
const userTrend = ref<UserUsageTrendPoint[]>([])
const userTrendMetric = ref<'tokens' | 'actual_cost'>('tokens')
const rankingItems = ref<UserSpendingRankingItem[]>([])
const rankingTotalActualCost = ref(0)
const rankingTotalRequests = ref(0)
const rankingTotalTokens = ref(0)
let chartLoadSeq = 0
let usersTrendLoadSeq = 0
let rankingLoadSeq = 0
const rankingLimit = 12

// Helper function to format date in local timezone
const formatLocalDate = (date: Date): string => {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

const getLast24HoursRangeDates = (): { start: string; end: string } => {
  const end = new Date()
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
  return {
    start: formatLocalDate(start),
    end: formatLocalDate(end)
  }
}

// Date range
const granularity = ref<'day' | 'hour'>('hour')
const defaultRange = getLast24HoursRangeDates()
const startDate = ref(defaultRange.start)
const endDate = ref(defaultRange.end)

// Granularity options for Select component
const granularityOptions = computed(() => [
  { value: 'day', label: t('admin.dashboard.day') },
  { value: 'hour', label: t('admin.dashboard.hour') }
])

// Dark mode detection
const isDarkMode = computed(() => {
  return document.documentElement.classList.contains('dark')
})

// Chart colors
const chartColors = computed(() => ({
  text: isDarkMode.value ? '#e5e7eb' : '#374151',
  grid: isDarkMode.value ? '#374151' : '#e5e7eb'
}))

// Line chart options (for user trend chart)
const lineOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    intersect: false,
    mode: 'index' as const
  },
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        color: chartColors.value.text,
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 15,
        font: {
          size: 11
        }
      }
    },
    tooltip: {
      itemSort: (a: any, b: any) => {
        const aValue = typeof a?.raw === 'number' ? a.raw : Number(a?.parsed?.y ?? 0)
        const bValue = typeof b?.raw === 'number' ? b.raw : Number(b?.parsed?.y ?? 0)
        return bValue - aValue
      },
      callbacks: {
        label: (context: any) => {
          return `${context.dataset.label}: ${formatUserTrendValue(Number(context.raw))}`
        }
      }
    }
  },
  scales: {
    x: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 10
        }
      }
    },
    y: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 10
        },
        callback: (value: string | number) => formatUserTrendValue(Number(value))
      }
    }
  }
}))

// User trend chart data
const userTrendChartData = computed(() => {
  if (!userTrend.value?.length) return null

  const getDisplayName = (point: UserUsageTrendPoint): string => {
    const username = point.username?.trim()
    if (username) {
      return username
    }

    const email = point.email?.trim()
    if (email) {
      return email
    }

    return t('admin.redeem.userPrefix', { id: point.user_id })
  }

  // Group by user_id to avoid merging different users with the same display name
  const userGroups = new Map<number, { name: string; data: Map<string, number> }>()
  const allDates = new Set<string>()

  userTrend.value.forEach((point) => {
    allDates.add(point.date)
    const key = point.user_id
    if (!userGroups.has(key)) {
      userGroups.set(key, { name: getDisplayName(point), data: new Map() })
    }
    userGroups.get(key)!.data.set(point.date, userTrendMetric.value === 'tokens' ? point.tokens : point.actual_cost)
  })

  const sortedDates = Array.from(allDates).sort()
  const colors = [
    '#3b82f6',
    '#10b981',
    '#f59e0b',
    '#ef4444',
    '#8b5cf6',
    '#ec4899',
    '#14b8a6',
    '#f97316',
    '#6366f1',
    '#84cc16',
    '#06b6d4',
    '#a855f7'
  ]

  const datasets = Array.from(userGroups.values()).map((group, idx) => ({
    label: group.name,
    data: sortedDates.map((date) => group.data.get(date) || 0),
    borderColor: colors[idx % colors.length],
    backgroundColor: `${colors[idx % colors.length]}20`,
    fill: false,
    tension: 0.3
  }))

  return {
    labels: sortedDates,
    datasets
  }
})

// Format helpers
const formatUserTrendValue = (value: number): string =>
  userTrendMetric.value === 'tokens' ? formatTokens(value) : `$${formatCost(value)}`

const formatTokens = (value: number | undefined): string => {
  if (value === undefined || value === null) return '0'
  if (value >= 1_000_000_000) {
    return `${(value / 1_000_000_000).toFixed(2)}B`
  } else if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(2)}M`
  } else if (value >= 1_000) {
    return `${(value / 1_000).toFixed(2)}K`
  }
  return value.toLocaleString()
}

const toFiniteNumber = (value: unknown): number => {
  const numberValue = Number(value)
  return Number.isFinite(numberValue) ? numberValue : 0
}

const formatNumber = (value: number | null | undefined): string => {
  return toFiniteNumber(value).toLocaleString()
}

const formatCost = (value: number | null | undefined): string => {
  const safeValue = toFiniteNumber(value)
  if (safeValue >= 1000) {
    return (safeValue / 1000).toFixed(2) + 'K'
  } else if (safeValue >= 1) {
    return safeValue.toFixed(2)
  } else if (safeValue >= 0.01) {
    return safeValue.toFixed(3)
  }
  return safeValue.toFixed(4)
}

const formatDuration = (ms: number): string => {
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(2)}s`
  }
  return `${Math.round(ms)}ms`
}

const goToUserUsage = (item: UserSpendingRankingItem) => {
  void router.push({
    path: '/admin/usage',
    query: {
      user_id: String(item.user_id),
      start_date: startDate.value,
      end_date: endDate.value
    }
  })
}

// Date range change handler
const onDateRangeChange = (range: {
  startDate: string
  endDate: string
  preset: string | null
}) => {
  // Auto-select granularity based on date range
  const start = new Date(range.startDate)
  const end = new Date(range.endDate)
  const daysDiff = Math.ceil((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24))

  // If range is 1 day, use hourly granularity
  if (daysDiff <= 1) {
    granularity.value = 'hour'
  } else {
    granularity.value = 'day'
  }

  loadChartData()
}

const onTrendModelChange = (model: string) => {
  trendModel.value = model
  trendData.value = []
  void loadDashboardSnapshot(false)
}

// Load data
const loadDashboardSnapshot = async (includeStats: boolean) => {
  const currentSeq = ++chartLoadSeq
  if (includeStats && !stats.value) {
    loading.value = true
  }
  chartsLoading.value = true
  try {
    const response = await adminAPI.dashboard.getSnapshotV2({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      include_stats: includeStats,
      include_trend: true,
      model: trendModel.value || undefined,
      include_model_stats: true,
      include_group_stats: false,
      include_users_trend: false
    })
    if (currentSeq !== chartLoadSeq) return
    if (includeStats && response.stats) {
      stats.value = response.stats
    }
    trendData.value = response.trend || []
    modelStats.value = response.models || []
  } catch (error) {
    if (currentSeq !== chartLoadSeq) return
    appStore.showError(t('admin.dashboard.failedToLoad'))
    console.error('Error loading dashboard snapshot:', error)
  } finally {
    if (currentSeq === chartLoadSeq) {
      loading.value = false
      chartsLoading.value = false
    }
  }
}

const setUserTrendMetric = (metric: 'tokens' | 'actual_cost') => {
  if (userTrendMetric.value === metric) return
  userTrendMetric.value = metric
  userTrend.value = []
  loadUsersTrend()
}

const loadUsersTrend = async () => {
  const currentSeq = ++usersTrendLoadSeq
  userTrendLoading.value = true
  try {
    const response = await adminAPI.dashboard.getUserUsageTrend({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      limit: 12,
      metric: userTrendMetric.value
    })
    if (currentSeq !== usersTrendLoadSeq) return
    userTrend.value = response.trend || []
  } catch (error) {
    if (currentSeq !== usersTrendLoadSeq) return
    console.error('Error loading users trend:', error)
    userTrend.value = []
  } finally {
    if (currentSeq === usersTrendLoadSeq) {
      userTrendLoading.value = false
    }
  }
}

const loadUserSpendingRanking = async () => {
  const currentSeq = ++rankingLoadSeq
  rankingLoading.value = true
  rankingError.value = false
  try {
    const response = await adminAPI.dashboard.getUserSpendingRanking({
      start_date: startDate.value,
      end_date: endDate.value,
      limit: rankingLimit
    })
    if (currentSeq !== rankingLoadSeq) return
    rankingItems.value = response.ranking || []
    rankingTotalActualCost.value = response.total_actual_cost || 0
    rankingTotalRequests.value = response.total_requests || 0
    rankingTotalTokens.value = response.total_tokens || 0
  } catch (error) {
    if (currentSeq !== rankingLoadSeq) return
    console.error('Error loading user spending ranking:', error)
    rankingItems.value = []
    rankingTotalActualCost.value = 0
    rankingTotalRequests.value = 0
    rankingTotalTokens.value = 0
    rankingError.value = true
  } finally {
    if (currentSeq === rankingLoadSeq) {
      rankingLoading.value = false
    }
  }
}

const loadDashboardStats = async () => {
  await Promise.all([
    loadDashboardSnapshot(true),
    loadUsersTrend(),
    loadUserSpendingRanking()
  ])
}

const loadChartData = async () => {
  await Promise.all([
    loadDashboardSnapshot(false),
    loadUsersTrend(),
    loadUserSpendingRanking()
  ])
}

onMounted(() => {
  void refreshBatchImageAccess()
  loadDashboardStats()
  void loadPerformance()
  void loadModelSummary()
  modelSummaryTimer = setInterval(() => { void loadModelSummary() }, 60_000)
  document.addEventListener('visibilitychange', loadModelSummary)
  performanceTimer = setInterval(() => { void loadPerformance() }, 10_000)
  errorFreeTimer = setInterval(() => {
    if (!performanceError.value) performanceClock.value = Date.now()
  }, 1000)
  document.addEventListener('visibilitychange', loadPerformance)
})
onUnmounted(() => {
  performanceDisposed = true
  if (modelSummaryTimer) clearInterval(modelSummaryTimer)
  modelSummaryController?.abort()
  document.removeEventListener('visibilitychange', loadModelSummary)
  if (performanceTimer) clearInterval(performanceTimer)
  if (errorFreeTimer) clearInterval(errorFreeTimer)
  performanceController?.abort()
  document.removeEventListener('visibilitychange', loadPerformance)
})

usePersistedFilters('admin/DashboardView', { startDate, endDate, granularity, userTrendMetric, trendModel })
</script>

<style scoped>
</style>
