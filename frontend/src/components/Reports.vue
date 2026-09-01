<template>
  <v-container fluid class="pa-4">
    <v-card rounded="lg" class="report-dashboard">
      <v-card-title class="d-flex flex-wrap align-center justify-space-between ga-3 pa-5">
        <div>
          <div class="text-h5 font-weight-bold">Spending Overview</div>
          <div class="text-body-2 text-medium-emphasis">
            Understand how much you spent and which expenses had the greatest impact.
          </div>
        </div>

        <div class="d-flex align-center ga-1">
          <v-btn
            icon="mdi-chevron-left"
            variant="text"
            size="small"
            aria-label="Previous month"
            title="Previous month"
            @click="changeMonth(-1)"
          ></v-btn>
          <v-btn
            prepend-icon="mdi-calendar-today"
            variant="tonal"
            size="small"
            @click="goToCurrentMonth"
          >
            Current Month
          </v-btn>
          <v-btn
            icon="mdi-chevron-right"
            variant="text"
            size="small"
            aria-label="Next month"
            title="Next month"
            @click="changeMonth(1)"
          ></v-btn>
        </div>
      </v-card-title>

      <v-divider></v-divider>

      <v-card-text class="pa-5">
        <v-sheet color="grey-lighten-4" rounded="lg" class="period-filter pa-3">
          <v-row dense align="center">
            <v-col cols="12" sm="5" md="3">
              <v-text-field
                v-model="startDate"
                label="From"
                type="date"
                density="compact"
                variant="outlined"
                bg-color="surface"
                hide-details
              ></v-text-field>
            </v-col>
            <v-col cols="12" sm="5" md="3">
              <v-text-field
                v-model="endDate"
                label="To"
                type="date"
                density="compact"
                variant="outlined"
                bg-color="surface"
                hide-details
              ></v-text-field>
            </v-col>
            <v-col cols="12" sm="2" md="2">
              <v-btn color="primary" block height="40" :disabled="!hasValidRange" @click="loadReport">
                Apply Period
              </v-btn>
            </v-col>
          </v-row>
        </v-sheet>

        <v-progress-linear v-if="loading" indeterminate color="primary" class="mt-4"></v-progress-linear>
        <v-alert v-if="errorMessage" type="error" variant="tonal" density="compact" class="mt-4">
          {{ errorMessage }}
        </v-alert>

        <v-row dense class="mt-4">
          <v-col v-for="summary in summaryCards" :key="summary.title" cols="12" sm="6" md="3">
            <v-card :color="summary.color" variant="tonal" class="summary-card pa-4 fill-height">
              <div class="d-flex align-start justify-space-between">
                <div>
                  <div class="text-caption text-uppercase font-weight-medium">{{ summary.title }}</div>
                  <div class="text-h5 font-weight-bold mt-1">{{ summary.value }}</div>
                  <div class="text-caption text-medium-emphasis mt-1">{{ summary.caption }}</div>
                </div>
                <v-avatar :color="summary.color" variant="tonal" size="42">
                  <v-icon :icon="summary.icon"></v-icon>
                </v-avatar>
              </div>
            </v-card>
          </v-col>
        </v-row>

        <v-alert
          v-if="!loading && report.transaction_count === 0"
          type="info"
          variant="tonal"
          class="mt-4"
        >
          No active expenses were found in this period.
        </v-alert>

        <v-row v-else class="mt-1" align="stretch">
          <v-col cols="12" lg="8">
            <v-card variant="outlined" rounded="lg" class="chart-card fill-height">
              <v-card-title class="d-flex flex-wrap align-center justify-space-between ga-2 px-4 pt-4">
                <div>
                  <div class="text-h6 font-weight-bold">Spending Distribution</div>
                  <div class="text-caption text-medium-emphasis">
                    Ranked from the largest expense to the smallest.
                  </div>
                </div>
                <v-chip size="small" color="primary" variant="tonal">
                  {{ selectedDimensionTitle }}
                </v-chip>
              </v-card-title>

              <v-tabs v-model="selectedDimension" density="compact" class="px-2 mt-2" show-arrows>
                <v-tab v-for="dimension in dimensions" :key="dimension.value" :value="dimension.value">
                  {{ dimension.title }}
                </v-tab>
              </v-tabs>
              <v-divider></v-divider>

              <v-card-text class="pa-4">
                <v-alert
                  v-if="selectedDimension === 'by_tag'"
                  type="info"
                  variant="tonal"
                  density="compact"
                  class="mb-4"
                >
                  A transaction with multiple tags appears in every matching tag, so tag shares are not additive.
                </v-alert>

                <div class="bar-chart" role="img" :aria-label="`Expenses grouped by ${selectedDimensionTitle}`">
                  <div
                    v-for="(group, index) in chartGroups"
                    :key="group.name.toLowerCase()"
                    class="bar-chart-row"
                  >
                    <div class="d-flex align-center justify-space-between ga-3 mb-1">
                      <span class="text-body-2 font-weight-medium text-truncate">{{ group.name }}</span>
                      <div class="text-right text-no-wrap">
                        <span class="text-body-2 font-weight-bold">{{ formatMoney(group.total_amount_cents) }}</span>
                        <span class="text-caption text-medium-emphasis ms-2">
                          {{ formatPercentage(group.percentage_of_expenses) }}
                        </span>
                      </div>
                    </div>
                    <div class="bar-chart-track">
                      <div
                        class="bar-chart-fill"
                        :style="{
                          width: `${chartBarWidth(group)}%`,
                          backgroundColor: chartColors[index % chartColors.length],
                        }"
                      ></div>
                    </div>
                  </div>
                </div>
              </v-card-text>
            </v-card>
          </v-col>

          <v-col cols="12" lg="4" class="d-flex flex-column ga-4">
            <v-card variant="outlined" rounded="lg" class="chart-card">
              <v-card-title class="px-4 pt-4">
                <div class="text-h6 font-weight-bold">Payment Status</div>
                <div class="text-caption text-medium-emphasis">Paid versus pending expenses.</div>
              </v-card-title>
              <v-card-text class="pa-4">
                <div class="payment-chart-layout">
                  <div class="donut-chart" :style="donutStyle" role="img" :aria-label="paymentChartLabel">
                    <div class="donut-center">
                      <div class="text-h5 font-weight-bold">{{ formatPercentage(paidPercentage) }}</div>
                      <div class="text-caption text-medium-emphasis">Paid</div>
                    </div>
                  </div>

                  <div class="payment-legend">
                    <div class="legend-row">
                      <span class="legend-dot legend-dot-paid"></span>
                      <div>
                        <div class="text-caption text-medium-emphasis">Paid</div>
                        <div class="font-weight-bold">{{ formatMoney(report.paid_expenses_cents) }}</div>
                      </div>
                    </div>
                    <div class="legend-row">
                      <span class="legend-dot legend-dot-pending"></span>
                      <div>
                        <div class="text-caption text-medium-emphasis">Pending</div>
                        <div class="font-weight-bold">{{ formatMoney(report.pending_expenses_cents) }}</div>
                      </div>
                    </div>
                  </div>
                </div>
              </v-card-text>
            </v-card>

            <v-card variant="outlined" rounded="lg" class="insights-card flex-grow-1">
              <v-card-title class="px-4 pt-4">
                <div class="text-h6 font-weight-bold">Key Insights</div>
              </v-card-title>
              <v-list density="compact" bg-color="transparent" class="px-2 pb-3">
                <v-list-item
                  v-for="insight in insightItems"
                  :key="insight.title"
                  :prepend-icon="insight.icon"
                  :title="insight.title"
                  :subtitle="insight.value"
                ></v-list-item>
              </v-list>
            </v-card>
          </v-col>
        </v-row>

        <v-card v-if="report.transaction_count > 0" variant="outlined" rounded="lg" class="mt-4">
          <v-card-title class="d-flex flex-wrap align-center justify-space-between ga-2 px-4 pt-4">
            <div>
              <div class="text-h6 font-weight-bold">Detailed Breakdown</div>
              <div class="text-caption text-medium-emphasis">
                Exact paid, pending, total, and share values for each {{ selectedDimensionTitle.toLowerCase() }}.
              </div>
            </div>
          </v-card-title>

          <div class="report-table mt-2">
            <v-table hover>
              <thead>
                <tr>
                  <th>{{ selectedDimensionTitle }}</th>
                  <th class="text-right">Transactions</th>
                  <th class="text-right">Paid</th>
                  <th class="text-right">Pending</th>
                  <th class="text-right">Total</th>
                  <th style="min-width: 150px;">Share</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="group in selectedGroups" :key="group.name.toLowerCase()">
                  <td class="font-weight-medium">{{ group.name }}</td>
                  <td class="text-right">{{ group.transaction_count }}</td>
                  <td class="text-right">{{ formatMoney(group.paid_amount_cents) }}</td>
                  <td class="text-right">{{ formatMoney(group.pending_amount_cents) }}</td>
                  <td class="text-right font-weight-bold">{{ formatMoney(group.total_amount_cents) }}</td>
                  <td>
                    <div class="d-flex align-center ga-2">
                      <v-progress-linear
                        :model-value="Math.min(Number(group.percentage_of_expenses) || 0, 100)"
                        color="primary"
                        rounded
                      ></v-progress-linear>
                      <span class="text-caption share-value">{{ formatPercentage(group.percentage_of_expenses) }}</span>
                    </div>
                  </td>
                </tr>
              </tbody>
            </v-table>
          </div>
        </v-card>
      </v-card-text>
    </v-card>
  </v-container>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue';
import { GetSpendingReport } from '../../wailsjs/go/main/App';
import { formatCurrencyFromCents } from '../utils/currency';

const props = defineProps({
  currencyCode: {
    type: String,
    default: 'USD',
  },
  refreshKey: {
    type: Number,
    default: 0,
  },
});

const dimensions = [
  { value: 'by_category', title: 'Category', heading: 'Category' },
  { value: 'by_subcategory', title: 'Subcategory', heading: 'Subcategory' },
  { value: 'by_payment_method', title: 'Payment Method', heading: 'Payment Method' },
  { value: 'by_tag', title: 'Tags', heading: 'Tag' },
];

const currentRange = monthRange(new Date());
const startDate = ref(currentRange.startDate);
const endDate = ref(currentRange.endDate);
const selectedDimension = ref('by_category');
const loading = ref(false);
const errorMessage = ref('');
const report = ref(emptyReport(currentRange.startDate, currentRange.endDate));

const hasValidRange = computed(() => (
  /^\d{4}-\d{2}-\d{2}$/.test(startDate.value)
  && /^\d{4}-\d{2}-\d{2}$/.test(endDate.value)
  && startDate.value <= endDate.value
));

const selectedGroups = computed(() => report.value[selectedDimension.value] || []);
const selectedDimensionTitle = computed(() => (
  dimensions.find(dimension => dimension.value === selectedDimension.value)?.heading || 'Group'
));

const chartColors = [
  '#1565c0',
  '#00897b',
  '#f9a825',
  '#8e24aa',
  '#e53935',
  '#5e35b1',
  '#43a047',
  '#fb8c00',
];

const chartGroups = computed(() => selectedGroups.value.slice(0, 8));
const largestGroup = computed(() => selectedGroups.value[0] || null);
const totalExpenses = computed(() => Number(report.value.total_expenses_cents) || 0);
const paidPercentage = computed(() => (
  totalExpenses.value > 0
    ? (Number(report.value.paid_expenses_cents) || 0) / totalExpenses.value * 100
    : 0
));
const pendingPercentage = computed(() => (
  totalExpenses.value > 0
    ? (Number(report.value.pending_expenses_cents) || 0) / totalExpenses.value * 100
    : 0
));
const averageExpense = computed(() => (
  report.value.transaction_count > 0
    ? Math.round(totalExpenses.value / report.value.transaction_count)
    : 0
));
const donutStyle = computed(() => {
  if (totalExpenses.value <= 0) {
    return { background: '#e0e0e0' };
  }
  const paid = Math.min(Math.max(paidPercentage.value, 0), 100);
  return {
    background: `conic-gradient(#2e7d32 0 ${paid}%, #fb8c00 ${paid}% 100%)`,
  };
});
const paymentChartLabel = computed(() => (
  `${formatPercentage(paidPercentage.value)} paid and ${formatPercentage(pendingPercentage.value)} pending`
));
const insightItems = computed(() => [
  {
    title: `Largest ${selectedDimensionTitle.value}`,
    value: largestGroup.value
      ? `${largestGroup.value.name} · ${formatMoney(largestGroup.value.total_amount_cents)}`
      : 'No expense data',
    icon: 'mdi-chart-timeline-variant-shimmer',
  },
  {
    title: 'Average per Transaction',
    value: formatMoney(averageExpense.value),
    icon: 'mdi-calculator-variant-outline',
  },
  {
    title: 'Payment Completion',
    value: `${formatPercentage(paidPercentage.value)} of expenses paid`,
    icon: 'mdi-check-circle-outline',
  },
]);

const summaryCards = computed(() => [
  {
    title: 'Total Expenses',
    value: formatMoney(report.value.total_expenses_cents),
    color: 'error',
    icon: 'mdi-cash-minus',
    caption: 'Paid and pending expenses',
  },
  {
    title: 'Paid',
    value: formatMoney(report.value.paid_expenses_cents),
    color: 'success',
    icon: 'mdi-check-circle-outline',
    caption: `${formatPercentage(paidPercentage.value)} of total expenses`,
  },
  {
    title: 'Pending',
    value: formatMoney(report.value.pending_expenses_cents),
    color: 'warning',
    icon: 'mdi-clock-alert-outline',
    caption: `${formatPercentage(pendingPercentage.value)} of total expenses`,
  },
  {
    title: 'Transactions',
    value: String(report.value.transaction_count || 0),
    color: 'primary',
    icon: 'mdi-receipt-text-outline',
    caption: 'Active expenses in this period',
  },
]);

function emptyReport(rangeStart, rangeEnd) {
  return {
    start_date: rangeStart,
    end_date: rangeEnd,
    total_expenses_cents: 0,
    paid_expenses_cents: 0,
    pending_expenses_cents: 0,
    transaction_count: 0,
    by_category: [],
    by_subcategory: [],
    by_payment_method: [],
    by_tag: [],
  };
}

function formatDate(date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function monthRange(date) {
  const start = new Date(date.getFullYear(), date.getMonth(), 1);
  const end = new Date(date.getFullYear(), date.getMonth() + 1, 0);
  return { startDate: formatDate(start), endDate: formatDate(end) };
}

function formatMoney(valueCents) {
  return formatCurrencyFromCents(valueCents, props.currencyCode);
}

function formatPercentage(value) {
  return `${Number(value || 0).toFixed(1)}%`;
}

function chartBarWidth(group) {
  const maximum = Math.max(
    ...chartGroups.value.map(item => Number(item.total_amount_cents) || 0),
    1,
  );
  return Math.max((Number(group.total_amount_cents) || 0) / maximum * 100, 2);
}

async function loadReport() {
  if (!hasValidRange.value) {
    errorMessage.value = 'Choose a valid date range.';
    return;
  }

  loading.value = true;
  errorMessage.value = '';
  try {
    report.value = await GetSpendingReport(startDate.value, endDate.value)
      || emptyReport(startDate.value, endDate.value);
  } catch (error) {
    console.error('Failed to load spending report:', error);
    report.value = emptyReport(startDate.value, endDate.value);
    errorMessage.value = 'Could not load the spending report for this period.';
  } finally {
    loading.value = false;
  }
}

function changeMonth(offset) {
  const reference = new Date(`${startDate.value}T00:00:00`);
  const baseDate = Number.isNaN(reference.getTime()) ? new Date() : reference;
  const range = monthRange(new Date(baseDate.getFullYear(), baseDate.getMonth() + offset, 1));
  startDate.value = range.startDate;
  endDate.value = range.endDate;
  loadReport();
}

function goToCurrentMonth() {
  const range = monthRange(new Date());
  startDate.value = range.startDate;
  endDate.value = range.endDate;
  loadReport();
}

onMounted(loadReport);
watch(() => props.refreshKey, loadReport);
</script>

<style scoped>
.report-dashboard {
  background: linear-gradient(180deg, rgba(245, 249, 255, 0.7) 0, #fff 260px);
  color: #212121;
}

.report-dashboard .text-medium-emphasis {
  color: rgba(33, 33, 33, 0.68) !important;
}

.period-filter {
  border: 1px solid rgba(0, 0, 0, 0.06);
}

.summary-card {
  min-height: 128px;
  transition: transform 160ms ease, box-shadow 160ms ease;
}

.summary-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.08);
}

.chart-card,
.insights-card {
  background-color: rgba(255, 255, 255, 0.88);
}

.bar-chart {
  display: flex;
  flex-direction: column;
  gap: 18px;
  min-height: 280px;
}

.bar-chart-row {
  min-width: 0;
}

.bar-chart-track {
  height: 14px;
  overflow: hidden;
  background: #edf1f5;
  border-radius: 999px;
}

.bar-chart-fill {
  height: 100%;
  min-width: 6px;
  border-radius: inherit;
  box-shadow: inset 0 -1px 0 rgba(0, 0, 0, 0.12);
  transition: width 300ms ease;
}

.payment-chart-layout {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: wrap;
  gap: 28px;
}

.donut-chart {
  width: 172px;
  height: 172px;
  display: grid;
  place-items: center;
  flex: 0 0 auto;
  border-radius: 50%;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
}

.donut-center {
  width: 112px;
  height: 112px;
  display: grid;
  place-content: center;
  text-align: center;
  background: white;
  border-radius: 50%;
}

.payment-legend {
  display: flex;
  flex-direction: column;
  gap: 18px;
  min-width: 120px;
}

.legend-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.legend-dot {
  width: 12px;
  height: 12px;
  flex: 0 0 auto;
  border-radius: 50%;
}

.legend-dot-paid {
  background: #2e7d32;
}

.legend-dot-pending {
  background: #fb8c00;
}

.report-table {
  overflow-x: auto;
}

.share-value {
  min-width: 45px;
  text-align: right;
}
</style>
