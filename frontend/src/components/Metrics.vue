<template>
  <v-container fluid>
    <v-sheet class="pa-4 border-sm" rounded="lg">
      <div class="d-flex flex-wrap align-center justify-space-between ga-3 mb-4">
        <h2>{{ periodTitle }} Metrics</h2>

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
            variant="text"
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
      </div>

      <v-progress-linear v-if="loading" indeterminate color="primary" class="mb-3"></v-progress-linear>
      <v-alert v-if="errorMessage" type="error" variant="tonal" density="compact" class="mb-3">
        {{ errorMessage }}
      </v-alert>

      <v-row dense>
        <v-col
          v-for="metric in metricCards"
          :key="metric.title"
          cols="12"
          sm="6"
          md="4"
          lg="2"
        >
          <v-card
            class="pa-2 fill-height rounded-lg"
            :color="metric.color"
            variant="tonal"
          >
            <v-card-title class="text-caption pa-0 mb-1">
              {{ metric.title }}
            </v-card-title>
            <v-card-text class="text-h6 pa-0 font-weight-bold">
              {{ metric.value }}
            </v-card-text>
          </v-card>
        </v-col>
      </v-row>

      <div v-if="categoryMetrics.length" class="mt-4">
        <div class="text-caption text-medium-emphasis mb-2">Category Breakdown</div>
        <div class="d-flex overflow-x-auto ga-2 pb-1">
          <v-chip
            v-for="category in categoryMetrics"
            :key="category.name"
            :color="category.type === 1 ? 'success' : 'warning'"
            variant="tonal"
          >
            {{ category.name }}: {{ formatMoney(category.total_amount_cents) }}
          </v-chip>
        </div>
      </div>
    </v-sheet>
  </v-container>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue';
import { GetFinancialMetrics } from '../../wailsjs/go/main/App';
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

const selectedMonth = ref(startOfMonth(new Date()));
const loading = ref(false);
const errorMessage = ref('');
const metrics = ref(emptyMetrics());

const periodTitle = computed(() => new Intl.DateTimeFormat('en-US', {
  month: 'long',
  year: 'numeric',
}).format(selectedMonth.value));

const categoryMetrics = computed(() => (
  metrics.value.categories || []
).filter(category => Number(category.total_amount_cents) !== 0));

const metricCards = computed(() => [
  {
    title: 'Received Income',
    value: formatMoney(metrics.value.received_income_cents),
    color: 'success',
  },
  {
    title: 'Paid Expenses',
    value: formatMoney(metrics.value.paid_expenses_cents),
    color: 'warning',
  },
  {
    title: 'Actual Balance',
    value: formatMoney(metrics.value.actual_balance_cents),
    color: metrics.value.actual_balance_cents >= 0 ? 'success' : 'error',
  },
  {
    title: 'Pending Expenses',
    value: formatMoney(metrics.value.pending_expenses_cents),
    color: 'warning',
  },
  {
    title: 'Expected Balance',
    value: formatMoney(metrics.value.expected_balance_cents),
    color: metrics.value.expected_balance_cents >= 0 ? 'primary' : 'error',
  },
  {
    title: 'Income Spent',
    value: metrics.value.has_received_income
      ? `${Number(metrics.value.income_spent_percentage).toFixed(1)}%`
      : 'N/A',
    color: 'surface-variant',
  },
]);

function emptyMetrics() {
  return {
    received_income_cents: 0,
    paid_expenses_cents: 0,
    pending_expenses_cents: 0,
    actual_balance_cents: 0,
    expected_balance_cents: 0,
    income_spent_percentage: 0,
    has_received_income: false,
    categories: [],
  };
}

function startOfMonth(date) {
  return new Date(date.getFullYear(), date.getMonth(), 1);
}

function formatDate(date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function getPeriodRange() {
  const start = startOfMonth(selectedMonth.value);
  const end = new Date(start.getFullYear(), start.getMonth() + 1, 0);
  return {
    startDate: formatDate(start),
    endDate: formatDate(end),
  };
}

function formatMoney(valueCents) {
  return formatCurrencyFromCents(valueCents, props.currencyCode);
}

async function loadMetrics() {
  const { startDate, endDate } = getPeriodRange();
  loading.value = true;
  errorMessage.value = '';

  try {
    metrics.value = await GetFinancialMetrics(startDate, endDate) || emptyMetrics();
  } catch (error) {
    console.error('Failed to load financial metrics:', error);
    metrics.value = emptyMetrics();
    errorMessage.value = 'Could not load the financial metrics for this period.';
  } finally {
    loading.value = false;
  }
}

function changeMonth(offset) {
  selectedMonth.value = new Date(
    selectedMonth.value.getFullYear(),
    selectedMonth.value.getMonth() + offset,
    1,
  );
  loadMetrics();
}

function goToCurrentMonth() {
  selectedMonth.value = startOfMonth(new Date());
  loadMetrics();
}

onMounted(loadMetrics);
watch(() => props.refreshKey, loadMetrics);
</script>
