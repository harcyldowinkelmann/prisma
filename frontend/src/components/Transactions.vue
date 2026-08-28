<template>
  <v-container fluid class="pa-4">
    <v-card>
      <v-card-title class="d-flex flex-wrap align-center ga-3 pa-4">
        <div>
          <div class="text-h6">All Transactions</div>
          <div class="text-caption text-medium-emphasis">
            {{ filteredTransactions.length }} of {{ transactions.length }} transactions
          </div>
        </div>
        <v-spacer></v-spacer>
        <v-btn variant="tonal" prepend-icon="mdi-bank-transfer-in" @click="statementDialogOpen = true">
          Import Statement
        </v-btn>
        <v-btn color="primary" prepend-icon="mdi-plus" @click="emit('request-add')">
          New Transaction
        </v-btn>
        <v-btn
          icon="mdi-refresh"
          variant="text"
          :loading="loading"
          aria-label="Refresh transactions"
          title="Refresh transactions"
          @click="loadData"
        ></v-btn>
      </v-card-title>

      <v-divider></v-divider>

      <v-card-text>
        <v-row dense>
          <v-col cols="12" md="4">
            <v-text-field
              v-model="search"
              prepend-inner-icon="mdi-magnify"
              label="Search transactions"
              variant="outlined"
              density="compact"
              clearable
              hide-details
            ></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="2">
            <v-select
              v-model="selectedCategory"
              :items="categoryOptions"
              label="Category"
              variant="outlined"
              density="compact"
              clearable
              hide-details
            ></v-select>
          </v-col>
          <v-col cols="12" sm="6" md="2">
            <v-select
              v-model="selectedReconciliation"
              :items="reconciliationOptions"
              item-title="title"
              item-value="value"
              label="Reconciliation"
              variant="outlined"
              density="compact"
              hide-details
            ></v-select>
          </v-col>
          <v-col cols="12" sm="6" md="2">
            <v-select
              v-model="selectedStatus"
              :items="statusOptions"
              item-title="title"
              item-value="value"
              label="Status"
              variant="outlined"
              density="compact"
              hide-details
            ></v-select>
          </v-col>
          <v-col cols="12" sm="6" md="2">
            <v-text-field
              v-model="startDate"
              label="From"
              type="date"
              variant="outlined"
              density="compact"
              hide-details
            ></v-text-field>
          </v-col>
          <v-col cols="12" sm="6" md="2">
            <v-text-field
              v-model="endDate"
              label="To"
              type="date"
              variant="outlined"
              density="compact"
              hide-details
            ></v-text-field>
          </v-col>
        </v-row>

        <div class="d-flex justify-end mt-3">
          <v-btn
            prepend-icon="mdi-filter-off"
            variant="text"
            size="small"
            :disabled="!hasActiveFilters"
            @click="clearFilters"
          >
            Clear Filters
          </v-btn>
        </div>

        <v-alert v-if="errorMessage" type="error" variant="tonal" density="compact" class="mt-3">
          {{ errorMessage }}
        </v-alert>
        <v-alert v-if="successMessage" type="success" variant="tonal" density="compact" class="mt-3" closable>
          {{ successMessage }}
        </v-alert>
      </v-card-text>

      <v-progress-linear v-if="loading" indeterminate color="primary"></v-progress-linear>
      <v-divider></v-divider>

      <v-table fixed-header height="500px">
        <thead>
          <tr>
            <th>Description</th>
            <th>Category</th>
            <th>Date</th>
            <th class="text-right">Amount</th>
            <th>Status</th>
            <th>Reconciliation</th>
            <th class="text-right">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="transaction in filteredTransactions"
            :key="transaction.id"
            :class="{ 'archived-transaction': !transaction.active }"
          >
            <td>
              <div class="font-weight-medium">{{ transaction.description }}</div>
              <div class="text-caption text-medium-emphasis transaction-details">
                {{ getTransactionDetails(transaction) }}
              </div>
            </td>
            <td>{{ transaction.category }}</td>
            <td>{{ formatDate(transaction.date) }}</td>
            <td class="text-right font-weight-medium">{{ formatMoney(transaction.amount_cents) }}</td>
            <td>
              <v-chip :color="getStatusColor(transaction)" size="small" variant="tonal">
                {{ getStatusLabel(transaction) }}
              </v-chip>
            </td>
            <td>
              <v-chip :color="transaction.reconciled ? 'primary' : 'grey'" size="small" variant="tonal">
                {{ transaction.reconciled ? 'Reconciled' : 'Unreconciled' }}
              </v-chip>
            </td>
            <td class="text-right text-no-wrap">
              <template v-if="transaction.active">
                <v-btn
                  :icon="transaction.reconciled ? 'mdi-bank-remove' : 'mdi-bank-check'"
                  variant="text"
                  size="small"
                  :color="transaction.reconciled ? 'grey' : 'primary'"
                  :aria-label="transaction.reconciled ? 'Mark as unreconciled' : 'Mark as reconciled'"
                  :title="transaction.reconciled ? 'Mark as unreconciled' : 'Mark as reconciled'"
                  @click="toggleReconciliation(transaction)"
                ></v-btn>
                <v-btn
                  icon="mdi-pencil"
                  variant="text"
                  size="small"
                  color="primary"
                  aria-label="Edit transaction"
                  title="Edit transaction"
                  @click="emit('request-edit', transaction)"
                ></v-btn>
                <v-btn
                  icon="mdi-archive-arrow-down-outline"
                  variant="text"
                  size="small"
                  color="error"
                  aria-label="Archive transaction"
                  title="Archive transaction"
                  @click="emit('request-archive', transaction.id)"
                ></v-btn>
              </template>
              <v-btn
                v-else
                icon="mdi-archive-arrow-up-outline"
                variant="text"
                size="small"
                color="success"
                aria-label="Restore transaction"
                title="Restore transaction"
                @click="emit('request-restore', transaction.id)"
              ></v-btn>
            </td>
          </tr>
          <tr v-if="!loading && filteredTransactions.length === 0">
            <td colspan="7" class="text-center text-medium-emphasis py-8">
              No transactions match the selected filters.
            </td>
          </tr>
        </tbody>
      </v-table>
    </v-card>

    <StatementImportDialog
      v-model="statementDialogOpen"
      :categories="categories"
      :currency-code="currencyCode"
      @imported="onStatementImported"
    />
  </v-container>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue';
import { GetCategories, GetTransactions, SetTransactionReconciled } from '../../wailsjs/go/main/App';
import { formatCurrencyFromCents } from '../utils/currency';
import StatementImportDialog from './StatementImportDialog.vue';

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

const emit = defineEmits(['request-add', 'request-edit', 'request-archive', 'request-restore', 'data-changed']);

const transactions = ref([]);
const categories = ref([]);
const search = ref('');
const selectedCategory = ref(null);
const selectedStatus = ref('all');
const selectedReconciliation = ref('all');
const startDate = ref('');
const endDate = ref('');
const loading = ref(false);
const errorMessage = ref('');
const successMessage = ref('');
const statementDialogOpen = ref(false);

const statusOptions = [
  { title: 'All', value: 'all' },
  { title: 'Paid', value: 'paid' },
  { title: 'Pending', value: 'pending' },
  { title: 'Archived', value: 'archived' },
];
const reconciliationOptions = [
  { title: 'All', value: 'all' },
  { title: 'Reconciled', value: 'reconciled' },
  { title: 'Unreconciled', value: 'unreconciled' },
];

const categoryOptions = computed(() => categories.value.map(category => category.name));

const hasActiveFilters = computed(() => Boolean(
  search.value
  || selectedCategory.value
  || selectedStatus.value !== 'all'
  || selectedReconciliation.value !== 'all'
  || startDate.value
  || endDate.value
));

const filteredTransactions = computed(() => {
  const normalizedSearch = (search.value || '').trim().toLowerCase();

  return transactions.value.filter(transaction => {
    if (selectedCategory.value && transaction.category !== selectedCategory.value) return false;
    if (startDate.value && transaction.date < startDate.value) return false;
    if (endDate.value && transaction.date > endDate.value) return false;

    if (selectedStatus.value === 'paid' && (!transaction.active || !transaction.is_paid)) return false;
    if (selectedStatus.value === 'pending' && (!transaction.active || transaction.is_paid)) return false;
    if (selectedStatus.value === 'archived' && transaction.active) return false;
    if (selectedReconciliation.value === 'reconciled' && !transaction.reconciled) return false;
    if (selectedReconciliation.value === 'unreconciled' && transaction.reconciled) return false;

    if (!normalizedSearch) return true;
    return [
      transaction.description,
      transaction.category,
      transaction.subcategory,
      transaction.payment_method,
      transaction.installments,
      transaction.tags,
    ].some(value => String(value || '').toLowerCase().includes(normalizedSearch));
  });
});

async function loadData() {
  loading.value = true;
  errorMessage.value = '';
  try {
    const [transactionResults, categoryResults] = await Promise.all([
      GetTransactions({ include_archived: true }),
      GetCategories(),
    ]);
    transactions.value = transactionResults || [];
    categories.value = categoryResults || [];
  } catch (error) {
    console.error('Failed to load all transactions:', error);
    errorMessage.value = 'Could not load the transaction list.';
  } finally {
    loading.value = false;
  }
}

function clearFilters() {
  search.value = '';
  selectedCategory.value = null;
  selectedStatus.value = 'all';
  selectedReconciliation.value = 'all';
  startDate.value = '';
  endDate.value = '';
}

function formatMoney(valueCents) {
  return formatCurrencyFromCents(valueCents, props.currencyCode);
}

function formatDate(isoDate) {
  if (!isoDate) return '';
  const [year, month, day] = isoDate.split('-');
  if (!year || !month || !day) return isoDate;
  return `${month}/${day}/${year}`;
}

function getStatusLabel(transaction) {
  if (!transaction.active) return 'Archived';
  return transaction.is_paid ? 'Paid' : 'Pending';
}

function getStatusColor(transaction) {
  if (!transaction.active) return 'grey';
  return transaction.is_paid ? 'success' : 'warning';
}

function getTransactionDetails(transaction) {
  return [
    transaction.subcategory,
    transaction.payment_method,
    transaction.installments,
    transaction.tags,
  ].filter(Boolean).join(' • ');
}

async function toggleReconciliation(transaction) {
  errorMessage.value = '';
  successMessage.value = '';
  try {
    await SetTransactionReconciled(transaction.id, !transaction.reconciled);
    await loadData();
    emit('data-changed');
  } catch (error) {
    console.error('Failed to update reconciliation state:', error);
    errorMessage.value = 'Could not update the reconciliation state.';
  }
}

async function onStatementImported(result) {
  successMessage.value = `Statement processed: ${result.imported_count} imported, ${result.reconciled_count} reconciled, ${result.skipped_count} skipped.`;
  await loadData();
  emit('data-changed');
}

onMounted(loadData);
watch(() => props.refreshKey, loadData);
</script>

<style scoped>
.archived-transaction {
  opacity: 0.65;
}

.transaction-details:empty {
  display: none;
}
</style>
