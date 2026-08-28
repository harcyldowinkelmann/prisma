<template>
  <v-container fluid class="pa-4">
    <v-card rounded="lg">
      <v-card-title class="d-flex flex-wrap align-center justify-space-between ga-3 pa-4">
        <div>
          <div class="text-h5">Spending Reports</div>
          <div class="text-body-2 text-medium-emphasis">See where your money went during the selected period.</div>
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
      </v-card-title>

      <v-divider></v-divider>

      <v-card-text class="pa-4">
        <v-row dense align="center">
          <v-col cols="12" sm="5" md="3">
            <v-text-field
              v-model="startDate"
              label="From"
              type="date"
              density="compact"
              hide-details
            ></v-text-field>
          </v-col>
          <v-col cols="12" sm="5" md="3">
            <v-text-field
              v-model="endDate"
              label="To"
              type="date"
              density="compact"
              hide-details
            ></v-text-field>
          </v-col>
          <v-col cols="12" sm="2" md="2">
            <v-btn color="primary" block :disabled="!hasValidRange" @click="loadReport">
              Apply
            </v-btn>
          </v-col>
        </v-row>

        <v-progress-linear v-if="loading" indeterminate color="primary" class="mt-4"></v-progress-linear>
        <v-alert v-if="errorMessage" type="error" variant="tonal" density="compact" class="mt-4">
          {{ errorMessage }}
        </v-alert>

        <v-row dense class="mt-4">
          <v-col v-for="summary in summaryCards" :key="summary.title" cols="12" sm="6" md="3">
            <v-card :color="summary.color" variant="tonal" class="pa-3 fill-height">
              <div class="text-caption">{{ summary.title }}</div>
              <div class="text-h6 font-weight-bold">{{ summary.value }}</div>
            </v-card>
          </v-col>
        </v-row>

        <v-tabs v-model="selectedDimension" class="mt-5" show-arrows>
          <v-tab v-for="dimension in dimensions" :key="dimension.value" :value="dimension.value">
            {{ dimension.title }}
          </v-tab>
        </v-tabs>

        <v-alert
          v-if="selectedDimension === 'by_tag'"
          type="info"
          variant="tonal"
          density="compact"
          class="mt-3"
        >
          Transactions with multiple tags are counted once in each matching tag, so tag rows are not additive.
        </v-alert>

        <v-alert
          v-if="!loading && report.transaction_count === 0"
          type="info"
          variant="tonal"
          class="mt-4"
        >
          No active expenses were found in this period.
        </v-alert>

        <div v-else class="report-table mt-3">
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

const summaryCards = computed(() => [
  {
    title: 'Total Expenses',
    value: formatMoney(report.value.total_expenses_cents),
    color: 'error',
  },
  {
    title: 'Paid',
    value: formatMoney(report.value.paid_expenses_cents),
    color: 'warning',
  },
  {
    title: 'Pending',
    value: formatMoney(report.value.pending_expenses_cents),
    color: 'orange-lighten-1',
  },
  {
    title: 'Transactions',
    value: String(report.value.transaction_count || 0),
    color: 'primary',
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
.report-table {
  overflow-x: auto;
}

.share-value {
  min-width: 45px;
  text-align: right;
}
</style>
