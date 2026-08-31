<template>
  <v-container fluid class="pa-4">
    <v-card rounded="lg">
      <v-card-title class="pa-4">
        <div class="text-h5">Planning</div>
        <div class="text-body-2 text-medium-emphasis">
          Plan category limits, installment purchases, and recurring transactions.
        </div>
      </v-card-title>

      <v-tabs v-model="activeSection" color="primary">
        <v-tab value="budgets">Monthly Budgets</v-tab>
        <v-tab value="recurring">Recurring Transactions</v-tab>
      </v-tabs>
      <v-divider></v-divider>

      <v-alert v-if="errorMessage" type="error" variant="tonal" density="compact" class="ma-4 mb-0">
        {{ errorMessage }}
      </v-alert>
      <v-alert v-if="successMessage" type="success" variant="tonal" density="compact" class="ma-4 mb-0">
        {{ successMessage }}
      </v-alert>
      <v-progress-linear v-if="loading" indeterminate color="primary"></v-progress-linear>

      <v-window v-model="activeSection">
        <v-window-item value="budgets">
          <v-card-text class="pa-4">
            <div class="d-flex flex-wrap align-center ga-2 mb-4">
              <div>
                <div class="text-h6">{{ budgetPeriodTitle }}</div>
                <div class="text-caption text-medium-emphasis">Active expenses, whether paid or pending, count toward the limit.</div>
              </div>
              <v-spacer></v-spacer>
              <v-btn icon="mdi-chevron-left" variant="text" aria-label="Previous month" @click="changeBudgetMonth(-1)"></v-btn>
              <v-btn prepend-icon="mdi-calendar-today" variant="text" @click="goToCurrentBudgetMonth">Current Month</v-btn>
              <v-btn icon="mdi-chevron-right" variant="text" aria-label="Next month" @click="changeBudgetMonth(1)"></v-btn>
            </div>

            <v-row dense class="mb-4">
              <v-col v-for="card in budgetCards" :key="card.title" cols="12" sm="4">
                <v-card :color="card.color" variant="tonal" class="pa-3 fill-height">
                  <div class="text-caption">{{ card.title }}</div>
                  <div class="text-h6 font-weight-bold">{{ card.value }}</div>
                </v-card>
              </v-col>
            </v-row>

            <v-alert v-if="expenseCategories.length === 0" type="info" variant="tonal">
              Add an expense category before creating a budget.
            </v-alert>

            <v-table v-else class="budget-table">
              <thead>
                <tr>
                  <th>Category</th>
                  <th style="min-width: 190px;">Monthly Limit</th>
                  <th class="text-right">Spent</th>
                  <th class="text-right">Remaining</th>
                  <th style="min-width: 180px;">Usage</th>
                  <th class="text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="category in expenseCategories" :key="category.uuid">
                  <td class="font-weight-medium">{{ category.name }}</td>
                  <td>
                    <v-text-field
                      v-model="budgetInputs[category.name]"
                      :prefix="currencySymbol"
                      type="number"
                      min="0.01"
                      step="0.01"
                      density="compact"
                      variant="outlined"
                      hide-details
                      placeholder="0.00"
                    ></v-text-field>
                  </td>
                  <td class="text-right">{{ hasBudget(category.name) ? formatMoney(summaryFor(category.name).spent_cents) : '—' }}</td>
                  <td
                    class="text-right font-weight-medium"
                    :class="summaryFor(category.name).over_budget ? 'text-error' : ''"
                  >
                    {{ hasBudget(category.name) ? formatMoney(summaryFor(category.name).remaining_cents) : '—' }}
                  </td>
                  <td>
                    <template v-if="hasBudget(category.name)">
                      <div class="d-flex align-center ga-2">
                        <v-progress-linear
                          :model-value="Math.min(summaryFor(category.name).percentage_used, 100)"
                          :color="summaryFor(category.name).over_budget ? 'error' : 'primary'"
                          rounded
                        ></v-progress-linear>
                        <span class="text-caption budget-percentage">{{ formatPercentage(summaryFor(category.name).percentage_used) }}</span>
                      </div>
                    </template>
                    <span v-else class="text-caption text-medium-emphasis">Not set</span>
                  </td>
                  <td class="text-right text-no-wrap">
                    <v-btn
                      icon="mdi-content-save"
                      size="small"
                      variant="text"
                      color="primary"
                      title="Save budget"
                      aria-label="Save budget"
                      @click="saveCategoryBudget(category.name)"
                    ></v-btn>
                    <v-btn
                      v-if="hasBudget(category.name)"
                      icon="mdi-delete-outline"
                      size="small"
                      variant="text"
                      color="error"
                      title="Delete budget"
                      aria-label="Delete budget"
                      @click="deleteCategoryBudget(category.name)"
                    ></v-btn>
                  </td>
                </tr>
              </tbody>
            </v-table>
          </v-card-text>
        </v-window-item>

        <v-window-item value="recurring">
          <v-card-text class="pa-4">
            <v-expansion-panels v-model="recurringFormPanel" class="mb-5">
              <v-expansion-panel value="form">
                <v-expansion-panel-title>
                  <div class="d-flex align-center">
                    <v-icon icon="mdi-calendar-sync" class="mr-2"></v-icon>
                    New Recurring Transaction
                  </div>
                </v-expansion-panel-title>
                <v-expansion-panel-text>
                  <v-alert type="info" variant="tonal" density="compact" class="mb-4">
                    Missing occurrences are created through the end of the current month. Existing occurrences are never duplicated.
                  </v-alert>
                  <v-row dense>
                    <v-col cols="12" md="6">
                      <v-text-field v-model="recurringForm.description" label="Description" variant="outlined"></v-text-field>
                    </v-col>
                    <v-col cols="12" md="3">
                      <v-text-field
                        v-model="recurringForm.amount"
                        :label="`Amount (${currencyCode})`"
                        :prefix="currencySymbol"
                        type="number"
                        min="0.01"
                        step="0.01"
                        variant="outlined"
                      ></v-text-field>
                    </v-col>
                    <v-col cols="12" md="3">
                      <v-select
                        v-model="recurringForm.frequency"
                        :items="frequencyOptions"
                        item-title="title"
                        item-value="value"
                        label="Frequency"
                        variant="outlined"
                      ></v-select>
                    </v-col>
                    <v-col cols="12" md="3">
                      <v-text-field v-model="recurringForm.startDate" label="Start Date" type="date" variant="outlined"></v-text-field>
                    </v-col>
                    <v-col cols="12" md="3">
                      <v-text-field v-model="recurringForm.endDate" label="End Date (optional)" type="date" variant="outlined" clearable></v-text-field>
                    </v-col>
                    <v-col cols="12" md="6">
                      <v-select
                        v-model="recurringForm.category"
                        :items="categories"
                        item-title="name"
                        item-value="name"
                        label="Category"
                        variant="outlined"
                      ></v-select>
                    </v-col>
                    <v-col cols="12" md="4">
                      <v-select v-model="recurringForm.subcategory" :items="subcategories" label="Subcategory" variant="outlined" clearable></v-select>
                    </v-col>
                    <v-col cols="12" md="4">
                      <v-select v-model="recurringForm.paymentMethod" :items="paymentMethods" label="Payment Method" variant="outlined" clearable></v-select>
                    </v-col>
                    <v-col cols="12" md="4">
                      <v-combobox v-model="recurringForm.tags" :items="tags" label="Tags" multiple chips variant="outlined"></v-combobox>
                    </v-col>
                    <v-col cols="12">
                      <v-checkbox
                        v-model="recurringForm.isPaid"
                        label="Generate occurrences as paid"
                        color="primary"
                        hide-details
                      ></v-checkbox>
                    </v-col>
                  </v-row>
                  <div class="d-flex justify-end mt-3">
                    <v-btn color="primary" prepend-icon="mdi-content-save" :loading="loading" @click="createRecurringSchedule">
                      Save Recurring Rule
                    </v-btn>
                  </div>
                </v-expansion-panel-text>
              </v-expansion-panel>
            </v-expansion-panels>

            <div class="d-flex flex-wrap align-center ga-2 mb-3">
              <div>
                <div class="text-h6">Active Recurring Rules</div>
                <div class="text-caption text-medium-emphasis">Stopping a rule does not remove transactions already generated.</div>
              </div>
              <v-spacer></v-spacer>
              <v-btn variant="tonal" prepend-icon="mdi-refresh" :loading="loading" @click="generateCurrentMonth">
                Generate Current Month
              </v-btn>
            </div>

            <v-alert v-if="recurringSchedules.length === 0" type="info" variant="tonal">
              No active recurring rules.
            </v-alert>
            <v-table v-else>
              <thead>
                <tr>
                  <th>Description</th>
                  <th>Category</th>
                  <th>Frequency</th>
                  <th>Period</th>
                  <th class="text-right">Amount</th>
                  <th class="text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="schedule in recurringSchedules" :key="schedule.uuid">
                  <td>
                    <div class="font-weight-medium">{{ schedule.description }}</div>
                    <div class="text-caption text-medium-emphasis">
                      {{ [schedule.subcategory, schedule.payment_method, schedule.tags].filter(Boolean).join(' • ') }}
                    </div>
                  </td>
                  <td>{{ schedule.category }}</td>
                  <td class="text-capitalize">{{ schedule.frequency }}</td>
                  <td>{{ schedule.start_date }} → {{ schedule.end_date || 'No end date' }}</td>
                  <td class="text-right">{{ formatMoney(schedule.amount_cents) }}</td>
                  <td class="text-right">
                    <v-btn
                      icon="mdi-stop-circle-outline"
                      color="error"
                      variant="text"
                      title="Stop recurring rule"
                      aria-label="Stop recurring rule"
                      @click="stopSchedule(schedule)"
                    ></v-btn>
                  </td>
                </tr>
              </tbody>
            </v-table>
          </v-card-text>
        </v-window-item>
      </v-window>
    </v-card>
  </v-container>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue';
import {
  AddRecurringSchedule,
  DeleteBudget,
  GenerateRecurringTransactions,
  GetBudgetSummaries,
  GetCategories,
  GetRecurringSchedules,
  GetSettings,
  SaveBudget,
  StopRecurringSchedule,
} from '../../wailsjs/go/main/App';
import { centsToDecimalString, formatCurrencyFromCents, getCurrencySymbol } from '../utils/currency';

const props = defineProps({
  currencyCode: { type: String, default: 'USD' },
  refreshKey: { type: Number, default: 0 },
});
const emit = defineEmits(['data-changed']);

const activeSection = ref('budgets');
const selectedBudgetMonth = ref(startOfMonth(new Date()));
const categories = ref([]);
const budgetSummaries = ref([]);
const budgetInputs = reactive({});
const recurringSchedules = ref([]);
const subcategories = ref([]);
const paymentMethods = ref([]);
const tags = ref([]);
const recurringFormPanel = ref(null);
const loading = ref(false);
const errorMessage = ref('');
const successMessage = ref('');

const recurringForm = reactive({
  description: '', amount: '', startDate: localDate(new Date()), endDate: '',
  frequency: 'monthly', category: '', subcategory: '', paymentMethod: '', tags: [], isPaid: false,
});

const frequencyOptions = [
  { title: 'Weekly', value: 'weekly' },
  { title: 'Monthly', value: 'monthly' },
  { title: 'Yearly', value: 'yearly' },
];

const currencySymbol = computed(() => getCurrencySymbol(props.currencyCode));
const expenseCategories = computed(() => categories.value.filter(category => category.type === -1));
const budgetMonth = computed(() => localDate(selectedBudgetMonth.value).slice(0, 7));
const budgetPeriodTitle = computed(() => new Intl.DateTimeFormat('en-US', {
  month: 'long', year: 'numeric',
}).format(selectedBudgetMonth.value));
const totalBudgetCents = computed(() => budgetSummaries.value.reduce((total, item) => total + Number(item.limit_cents || 0), 0));
const totalSpentCents = computed(() => budgetSummaries.value.reduce((total, item) => total + Number(item.spent_cents || 0), 0));
const budgetCards = computed(() => [
  { title: 'Total Budget', value: formatMoney(totalBudgetCents.value), color: 'primary' },
  { title: 'Budgeted Spending', value: formatMoney(totalSpentCents.value), color: 'warning' },
  {
    title: 'Remaining',
    value: formatMoney(totalBudgetCents.value - totalSpentCents.value),
    color: totalSpentCents.value > totalBudgetCents.value && totalBudgetCents.value > 0 ? 'error' : 'success',
  },
]);

function startOfMonth(date) {
  return new Date(date.getFullYear(), date.getMonth(), 1);
}

function localDate(date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function endOfCurrentMonth() {
  const today = new Date();
  return localDate(new Date(today.getFullYear(), today.getMonth() + 1, 0));
}

function formatMoney(valueCents) {
  return formatCurrencyFromCents(valueCents, props.currencyCode);
}

function formatPercentage(value) {
  return `${Number(value || 0).toFixed(1)}%`;
}

function summaryFor(categoryName) {
  return budgetSummaries.value.find(item => item.category === categoryName) || {
    limit_cents: 0, spent_cents: 0, remaining_cents: 0, percentage_used: 0, over_budget: false,
  };
}

function hasBudget(categoryName) {
  return budgetSummaries.value.some(item => item.category === categoryName);
}

async function loadPlanningData() {
  loading.value = true;
  errorMessage.value = '';
  try {
    const [categoryResults, subcategoryResults, paymentResults, tagResults, scheduleResults] = await Promise.all([
      GetCategories(), GetSettings('subcategories'), GetSettings('payment_methods'), GetSettings('tags'), GetRecurringSchedules(),
    ]);
    categories.value = categoryResults || [];
    subcategories.value = (subcategoryResults || []).map(item => item.name);
    paymentMethods.value = (paymentResults || []).map(item => item.name);
    tags.value = (tagResults || []).map(item => item.name);
    recurringSchedules.value = scheduleResults || [];
    if (!recurringForm.category && categories.value.length) recurringForm.category = categories.value[0].name;
    await loadBudgets();
  } catch (error) {
    console.error('Failed to load planning data:', error);
    errorMessage.value = 'Could not load planning data.';
  } finally {
    loading.value = false;
  }
}

async function loadBudgets() {
  const results = await GetBudgetSummaries(budgetMonth.value);
  budgetSummaries.value = results || [];
  for (const category of expenseCategories.value) {
    const summary = summaryFor(category.name);
    budgetInputs[category.name] = hasBudget(category.name) ? centsToDecimalString(summary.limit_cents) : '';
  }
}

async function saveCategoryBudget(categoryName) {
  const amount = String(budgetInputs[categoryName] || '').trim();
  if (!/^(?:\d+|\d*\.\d{1,2})$/.test(amount) || /^0*(?:\.0{1,2})?$/.test(amount)) {
    errorMessage.value = 'Enter a positive budget with at most two decimal places.';
    return;
  }
  loading.value = true;
  errorMessage.value = '';
  try {
    await SaveBudget(budgetMonth.value, categoryName, amount);
    await loadBudgets();
    successMessage.value = `${categoryName} budget saved.`;
  } catch (error) {
    console.error('Failed to save budget:', error);
    errorMessage.value = 'Could not save the budget.';
  } finally {
    loading.value = false;
  }
}

async function deleteCategoryBudget(categoryName) {
  if (!confirm(`Delete the ${categoryName} budget for ${budgetPeriodTitle.value}?`)) return;
  loading.value = true;
  errorMessage.value = '';
  try {
    await DeleteBudget(budgetMonth.value, categoryName);
    await loadBudgets();
    successMessage.value = `${categoryName} budget deleted.`;
  } catch (error) {
    console.error('Failed to delete budget:', error);
    errorMessage.value = 'Could not delete the budget.';
  } finally {
    loading.value = false;
  }
}

async function changeBudgetMonth(offset) {
  selectedBudgetMonth.value = new Date(
    selectedBudgetMonth.value.getFullYear(), selectedBudgetMonth.value.getMonth() + offset, 1,
  );
  await loadBudgets();
}

async function goToCurrentBudgetMonth() {
  selectedBudgetMonth.value = startOfMonth(new Date());
  await loadBudgets();
}

async function createRecurringSchedule() {
  const amount = String(recurringForm.amount || '').trim();
  const validAmount = /^(?:\d+|\d*\.\d{1,2})$/.test(amount) && !/^0*(?:\.0{1,2})?$/.test(amount);
  if (!recurringForm.description.trim() || !validAmount || !recurringForm.startDate || !recurringForm.category) {
    errorMessage.value = 'Description, amount, start date, and category are required.';
    return;
  }
  if (recurringForm.endDate && recurringForm.endDate < recurringForm.startDate) {
    errorMessage.value = 'End date must not be before the start date.';
    return;
  }
  loading.value = true;
  errorMessage.value = '';
  try {
    await AddRecurringSchedule(
      recurringForm.description.trim(), amount, recurringForm.startDate, recurringForm.endDate || '',
      recurringForm.frequency, recurringForm.category, recurringForm.subcategory || '',
      recurringForm.paymentMethod || '', recurringForm.tags.join(', '), recurringForm.isPaid,
    );
    resetRecurringForm();
    recurringSchedules.value = await GetRecurringSchedules() || [];
    successMessage.value = 'Recurring rule saved and current-month occurrences generated.';
    emit('data-changed');
  } catch (error) {
    console.error('Failed to save recurring rule:', error);
    errorMessage.value = 'Could not save the recurring rule.';
  } finally {
    loading.value = false;
  }
}

async function stopSchedule(schedule) {
  if (!confirm(`Stop the recurring rule “${schedule.description}”? Existing transactions will remain.`)) return;
  loading.value = true;
  errorMessage.value = '';
  try {
    await StopRecurringSchedule(schedule.uuid);
    recurringSchedules.value = await GetRecurringSchedules() || [];
    successMessage.value = 'Recurring rule stopped.';
  } catch (error) {
    console.error('Failed to stop recurring rule:', error);
    errorMessage.value = 'Could not stop the recurring rule.';
  } finally {
    loading.value = false;
  }
}

async function generateCurrentMonth() {
  loading.value = true;
  errorMessage.value = '';
  try {
    const generatedCount = await GenerateRecurringTransactions(endOfCurrentMonth());
    successMessage.value = `${generatedCount} missing occurrence${generatedCount === 1 ? '' : 's'} generated.`;
    if (generatedCount > 0) emit('data-changed');
  } catch (error) {
    console.error('Failed to generate recurring transactions:', error);
    errorMessage.value = 'Could not generate recurring transactions.';
  } finally {
    loading.value = false;
  }
}

function resetRecurringForm() {
  recurringForm.description = '';
  recurringForm.amount = '';
  recurringForm.startDate = localDate(new Date());
  recurringForm.endDate = '';
  recurringForm.frequency = 'monthly';
  recurringForm.category = categories.value[0]?.name || '';
  recurringForm.subcategory = '';
  recurringForm.paymentMethod = '';
  recurringForm.tags = [];
  recurringForm.isPaid = false;
  recurringFormPanel.value = null;
}

onMounted(loadPlanningData);
watch(() => props.refreshKey, loadBudgets);
</script>

<style scoped>
.budget-table {
  overflow-x: auto;
}

.budget-percentage {
  min-width: 50px;
  text-align: right;
}
</style>
