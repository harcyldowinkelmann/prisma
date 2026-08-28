<template>
  <v-dialog :model-value="modelValue" max-width="1200" persistent @update:model-value="emit('update:modelValue', $event)">
    <v-card>
      <v-card-title class="d-flex align-center pa-4">
        <v-icon icon="mdi-bank-transfer-in" class="mr-2"></v-icon>
        Import Bank Statement
        <v-spacer></v-spacer>
        <v-btn icon="mdi-close" variant="text" aria-label="Close" @click="closeDialog"></v-btn>
      </v-card-title>
      <v-divider></v-divider>

      <v-card-text class="pa-4">
        <v-alert type="info" variant="tonal" density="compact" class="mb-4">
          Review every row before importing. Prisma never changes financial data during the preview.
        </v-alert>

        <v-alert v-if="errorMessage" type="error" variant="tonal" density="compact" class="mb-4">
          {{ errorMessage }}
        </v-alert>

        <template v-if="stage === 'file'">
          <div class="text-h6 mb-3">1. Select the CSV file</div>
          <v-file-input
            v-model="selectedFile"
            label="Bank statement CSV"
            accept=".csv,text/csv,text/plain"
            prepend-icon="mdi-file-delimited-outline"
            variant="outlined"
            show-size
          ></v-file-input>
          <v-row dense>
            <v-col cols="12" sm="6">
              <v-select
                v-model="delimiter"
                :items="delimiterOptions"
                item-title="title"
                item-value="value"
                label="Column separator"
                variant="outlined"
              ></v-select>
            </v-col>
            <v-col cols="12" sm="6">
              <v-checkbox v-model="hasHeader" label="The first row contains column names"></v-checkbox>
            </v-col>
          </v-row>
        </template>

        <template v-else-if="stage === 'mapping'">
          <div class="d-flex align-center mb-3">
            <div class="text-h6">2. Map statement columns</div>
            <v-spacer></v-spacer>
            <v-chip size="small" variant="tonal">{{ inspection.headers.length }} columns detected</v-chip>
          </div>

          <v-row dense>
            <v-col cols="12" md="4">
              <v-select v-model="parseOptions.date_column" :items="columnItems" label="Date column" variant="outlined"></v-select>
            </v-col>
            <v-col cols="12" md="4">
              <v-select v-model="parseOptions.description_column" :items="columnItems" label="Description column" variant="outlined"></v-select>
            </v-col>
            <v-col cols="12" md="4">
              <v-select
                v-model="parseOptions.date_format"
                :items="dateFormatOptions"
                item-title="title"
                item-value="value"
                label="Date format"
                variant="outlined"
              ></v-select>
            </v-col>
          </v-row>

          <v-radio-group v-model="parseOptions.amount_mode" inline label="Amount columns" class="mb-2">
            <v-radio label="One signed amount column" value="signed"></v-radio>
            <v-radio label="Separate debit and credit columns" value="debit_credit"></v-radio>
          </v-radio-group>

          <v-alert v-if="parseOptions.amount_mode === 'signed'" type="info" variant="tonal" density="compact" class="mb-3">
            Positive values are imported as income. Negative values are imported as expenses.
          </v-alert>

          <v-row dense>
            <v-col v-if="parseOptions.amount_mode === 'signed'" cols="12" md="6">
              <v-select v-model="parseOptions.amount_column" :items="columnItems" label="Amount column" variant="outlined"></v-select>
            </v-col>
            <template v-else>
              <v-col cols="12" md="3">
                <v-select v-model="parseOptions.debit_column" :items="optionalColumnItems" label="Debit column" variant="outlined"></v-select>
              </v-col>
              <v-col cols="12" md="3">
                <v-select v-model="parseOptions.credit_column" :items="optionalColumnItems" label="Credit column" variant="outlined"></v-select>
              </v-col>
            </template>
            <v-col cols="12" md="6">
              <v-select
                v-model="parseOptions.decimal_separator"
                :items="decimalOptions"
                item-title="title"
                item-value="value"
                label="Decimal separator"
                variant="outlined"
              ></v-select>
            </v-col>
          </v-row>

          <div class="text-subtitle-1 font-weight-bold mb-2">Defaults for new transactions</div>
          <v-row dense>
            <v-col cols="12" md="6">
              <v-select
                v-model="importOptions.income_category"
                :items="incomeCategories"
                item-title="name"
                item-value="name"
                label="Income category"
                variant="outlined"
              ></v-select>
            </v-col>
            <v-col cols="12" md="6">
              <v-select
                v-model="importOptions.expense_category"
                :items="expenseCategories"
                item-title="name"
                item-value="name"
                label="Expense category"
                variant="outlined"
              ></v-select>
            </v-col>
            <v-col cols="12" md="4">
              <v-text-field v-model="importOptions.subcategory" label="Subcategory (optional)" variant="outlined"></v-text-field>
            </v-col>
            <v-col cols="12" md="4">
              <v-text-field v-model="importOptions.payment_method" label="Payment method (optional)" variant="outlined"></v-text-field>
            </v-col>
            <v-col cols="12" md="4">
              <v-text-field v-model="importOptions.tags" label="Tags (optional)" variant="outlined"></v-text-field>
            </v-col>
          </v-row>

          <div class="text-caption text-medium-emphasis mb-2">Sample rows</div>
          <div class="sample-table mb-3">
            <v-table density="compact">
              <thead><tr><th v-for="header in inspection.headers" :key="header">{{ header }}</th></tr></thead>
              <tbody>
                <tr v-for="(row, rowIndex) in inspection.sample_rows" :key="rowIndex">
                  <td v-for="(value, columnIndex) in row" :key="columnIndex">{{ value }}</td>
                </tr>
              </tbody>
            </v-table>
          </div>
        </template>

        <template v-else>
          <div class="d-flex flex-wrap align-center ga-2 mb-3">
            <div class="text-h6">3. Review and confirm</div>
            <v-spacer></v-spacer>
            <v-chip color="primary" variant="tonal">{{ actionableCount }} selected</v-chip>
            <v-chip v-if="duplicateCount" color="grey" variant="tonal">{{ duplicateCount }} duplicates</v-chip>
            <v-chip v-if="preview.errors.length" color="error" variant="tonal">{{ preview.errors.length }} rejected</v-chip>
          </div>

          <v-alert v-if="preview.errors.length" type="warning" variant="tonal" density="compact" class="mb-3">
            <div class="font-weight-bold mb-1">Rows requiring correction</div>
            <div v-for="rowError in preview.errors" :key="rowError.row_number">
              Row {{ rowError.row_number }}: {{ rowError.message }}
            </div>
          </v-alert>

          <div class="preview-table">
            <v-table fixed-header height="440px" density="compact">
              <thead>
                <tr>
                  <th>Row</th>
                  <th>Date</th>
                  <th>Description</th>
                  <th>Type</th>
                  <th class="text-right">Amount</th>
                  <th>Match</th>
                  <th style="min-width: 190px;">Action</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in preview.rows" :key="row.fingerprint" :class="{ 'duplicate-row': row.duplicate }">
                  <td>{{ row.row_number }}</td>
                  <td>{{ row.date }}</td>
                  <td>{{ row.description }}</td>
                  <td>
                    <v-chip :color="row.type === 1 ? 'success' : 'warning'" size="x-small" variant="tonal">
                      {{ row.type === 1 ? 'Income' : 'Expense' }}
                    </v-chip>
                  </td>
                  <td class="text-right">{{ formatMoney(row.amount_cents) }}</td>
                  <td>
                    <span v-if="row.duplicate">Already imported</span>
                    <span v-else-if="row.matched_reconciled">Already reconciled: {{ row.matched_description }}</span>
                    <span v-else-if="row.matched_transaction_id">{{ row.matched_description }}</span>
                    <span v-else class="text-medium-emphasis">No unique match</span>
                  </td>
                  <td>
                    <v-select
                      v-model="row.action"
                      :items="actionOptions(row)"
                      item-title="title"
                      item-value="value"
                      density="compact"
                      variant="outlined"
                      hide-details
                      :disabled="row.duplicate"
                    ></v-select>
                  </td>
                </tr>
              </tbody>
            </v-table>
          </div>
        </template>
      </v-card-text>

      <v-divider></v-divider>
      <v-card-actions class="pa-4">
        <v-btn v-if="stage !== 'file'" variant="text" :disabled="loading" @click="goBack">Back</v-btn>
        <v-spacer></v-spacer>
        <v-btn variant="text" :disabled="loading" @click="closeDialog">Cancel</v-btn>
        <v-btn
          v-if="stage === 'file'"
          color="primary"
          :loading="loading"
          :disabled="!selectedFile"
          @click="inspectFile"
        >
          Continue
        </v-btn>
        <v-btn
          v-else-if="stage === 'mapping'"
          color="primary"
          :loading="loading"
          :disabled="!mappingIsValid"
          @click="previewStatement"
        >
          Preview
        </v-btn>
        <v-btn
          v-else
          color="primary"
          :loading="loading"
          :disabled="actionableCount === 0 || !importCategoriesAreValid"
          @click="importStatement"
        >
          Confirm Import
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue';
import { ImportStatementRows, InspectStatementCSV, PreviewStatementCSV } from '../../wailsjs/go/main/App';
import { formatCurrencyFromCents } from '../utils/currency';

const props = defineProps({
  modelValue: Boolean,
  categories: { type: Array, default: () => [] },
  currencyCode: { type: String, default: 'USD' },
});
const emit = defineEmits(['update:modelValue', 'imported']);

const delimiterOptions = [
  { title: 'Detect automatically', value: 'auto' },
  { title: 'Comma (,)', value: 'comma' },
  { title: 'Semicolon (;)', value: 'semicolon' },
  { title: 'Tab', value: 'tab' },
];
const dateFormatOptions = [
  { title: 'YYYY-MM-DD', value: 'yyyy-mm-dd' },
  { title: 'MM/DD/YYYY', value: 'mm/dd/yyyy' },
  { title: 'DD/MM/YYYY', value: 'dd/mm/yyyy' },
  { title: 'MM-DD-YYYY', value: 'mm-dd-yyyy' },
  { title: 'DD-MM-YYYY', value: 'dd-mm-yyyy' },
];
const decimalOptions = [
  { title: 'Detect automatically', value: 'auto' },
  { title: 'Dot (1,234.56)', value: 'dot' },
  { title: 'Comma (1.234,56)', value: 'comma' },
];

const stage = ref('file');
const selectedFile = ref(null);
const fileContent = ref('');
const delimiter = ref('auto');
const hasHeader = ref(true);
const loading = ref(false);
const errorMessage = ref('');
const inspection = ref(emptyInspection());
const preview = ref({ rows: [], errors: [] });

const parseOptions = reactive({
  delimiter: 'auto', has_header: true, date_column: -1, description_column: -1,
  amount_mode: 'signed', amount_column: -1, debit_column: -1, credit_column: -1,
  date_format: 'yyyy-mm-dd', decimal_separator: 'auto',
});
const importOptions = reactive({
  income_category: '', expense_category: '', subcategory: '', payment_method: '', tags: '',
});

const incomeCategories = computed(() => props.categories.filter(category => category.type === 1));
const expenseCategories = computed(() => props.categories.filter(category => category.type === -1));
const columnItems = computed(() => inspection.value.headers.map((header, index) => ({ title: header, value: index })));
const optionalColumnItems = computed(() => [{ title: 'Not used', value: -1 }, ...columnItems.value]);
const actionableCount = computed(() => preview.value.rows.filter(row => row.action !== 'skip').length);
const duplicateCount = computed(() => preview.value.rows.filter(row => row.duplicate).length);
const importsIncome = computed(() => preview.value.rows.some(row => row.action === 'import' && row.type === 1));
const importsExpenses = computed(() => preview.value.rows.some(row => row.action === 'import' && row.type === -1));

const mappingIsValid = computed(() => (
  parseOptions.date_column >= 0
  && parseOptions.description_column >= 0
  && (
    (parseOptions.amount_mode === 'signed' && parseOptions.amount_column >= 0)
    || (parseOptions.amount_mode === 'debit_credit' && (parseOptions.debit_column >= 0 || parseOptions.credit_column >= 0))
  )
));
const importCategoriesAreValid = computed(() => (
  (!importsIncome.value || Boolean(importOptions.income_category))
  && (!importsExpenses.value || Boolean(importOptions.expense_category))
));

function emptyInspection() {
  return { headers: [], sample_rows: [] };
}

function initializeCategories() {
  if (!importOptions.income_category && incomeCategories.value.length) {
    importOptions.income_category = incomeCategories.value[0].name;
  }
  if (!importOptions.expense_category && expenseCategories.value.length) {
    importOptions.expense_category = expenseCategories.value[0].name;
  }
}

async function inspectFile() {
  loading.value = true;
  errorMessage.value = '';
  try {
    const file = Array.isArray(selectedFile.value) ? selectedFile.value[0] : selectedFile.value;
    if (!file) throw new Error('Select a CSV file.');
    fileContent.value = await file.text();
    inspection.value = await InspectStatementCSV(fileContent.value, delimiter.value, hasHeader.value);
    parseOptions.delimiter = inspection.value.delimiter;
    parseOptions.has_header = hasHeader.value;
    parseOptions.date_column = inspection.value.detected_date_column;
    parseOptions.description_column = inspection.value.detected_description_column;
    parseOptions.amount_column = inspection.value.detected_amount_column;
    parseOptions.debit_column = inspection.value.detected_debit_column;
    parseOptions.credit_column = inspection.value.detected_credit_column;
    parseOptions.date_format = inspection.value.detected_date_format || 'yyyy-mm-dd';
    parseOptions.amount_mode = parseOptions.amount_column >= 0 ? 'signed' : 'debit_credit';
    initializeCategories();
    stage.value = 'mapping';
  } catch (error) {
    console.error('Failed to inspect statement:', error);
    errorMessage.value = String(error);
  } finally {
    loading.value = false;
  }
}

async function previewStatement() {
  loading.value = true;
  errorMessage.value = '';
  try {
    preview.value = await PreviewStatementCSV(fileContent.value, { ...parseOptions });
    stage.value = 'preview';
  } catch (error) {
    console.error('Failed to preview statement:', error);
    errorMessage.value = String(error);
  } finally {
    loading.value = false;
  }
}

async function importStatement() {
  loading.value = true;
  errorMessage.value = '';
  try {
    const result = await ImportStatementRows(preview.value.rows, { ...importOptions });
    emit('imported', result);
    closeDialog();
  } catch (error) {
    console.error('Failed to import statement:', error);
    errorMessage.value = String(error);
  } finally {
    loading.value = false;
  }
}

function actionOptions(row) {
  if (row.duplicate) return [{ title: 'Skip duplicate', value: 'skip' }];
  const options = [];
  if (row.matched_transaction_id && !row.matched_reconciled) {
    options.push({ title: 'Reconcile existing', value: 'reconcile' });
  }
  options.push({ title: 'Import as new', value: 'import' });
  options.push({ title: 'Skip', value: 'skip' });
  return options;
}

function formatMoney(valueCents) {
  return formatCurrencyFromCents(valueCents, props.currencyCode);
}

function goBack() {
  errorMessage.value = '';
  stage.value = stage.value === 'preview' ? 'mapping' : 'file';
}

function closeDialog() {
  emit('update:modelValue', false);
}

function resetDialog() {
  stage.value = 'file';
  selectedFile.value = null;
  fileContent.value = '';
  delimiter.value = 'auto';
  hasHeader.value = true;
  errorMessage.value = '';
  inspection.value = emptyInspection();
  preview.value = { rows: [], errors: [] };
}

watch(() => props.modelValue, isOpen => {
  if (isOpen) {
    resetDialog();
    initializeCategories();
  }
});
watch(() => props.categories, initializeCategories, { deep: true });
</script>

<style scoped>
.sample-table,
.preview-table {
  overflow-x: auto;
  border: 1px solid rgba(0, 0, 0, 0.12);
  border-radius: 4px;
}

.duplicate-row {
  opacity: 0.6;
}
</style>
