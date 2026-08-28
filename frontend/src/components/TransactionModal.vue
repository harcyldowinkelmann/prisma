<template>
  <div>
    <v-snackbar
      v-model="snackbar.show"
      :color="snackbar.color"
      timeout="3000"
      location="top right"
    >
      {{ snackbar.text }}
      <template #actions>
        <v-btn variant="text" @click="snackbar.show = false">Close</v-btn>
      </template>
    </v-snackbar>

    <v-dialog
      :model-value="modelValue"
      max-width="650px"
      @update:model-value="handleDialogChange"
    >
      <v-card>
        <v-card-title>
          {{ isEditing ? 'Edit Transaction' : 'New Transaction' }}
        </v-card-title>

        <v-card-text>
          <v-container>
            <v-row>
              <v-col cols="12">
                <v-tooltip text="Briefly describe what this transaction is about (e.g., 'Walmart Groceries')" location="top">
                  <template #activator="{ props: tooltipProps }">
                    <v-text-field
                      v-bind="tooltipProps"
                      v-model="form.description"
                      label="Description"
                      required
                      variant="outlined"
                      autofocus
                    ></v-text-field>
                  </template>
                </v-tooltip>
              </v-col>

              <v-col cols="12" sm="6">
                <v-select
                  v-model="form.category"
                  :items="availableCategories"
                  label="Category"
                  variant="outlined"
                  required
                ></v-select>
              </v-col>

              <v-col cols="12" sm="6">
                <v-tooltip text="The total value of the transaction" location="top">
                  <template #activator="{ props: tooltipProps }">
                    <v-text-field
                      v-bind="tooltipProps"
                      v-model="form.amount"
                      :label="amountLabel"
                      :prefix="currencySymbol"
                      type="number"
                      min="0.01"
                      step="0.01"
                      variant="outlined"
                    ></v-text-field>
                  </template>
                </v-tooltip>
              </v-col>

              <v-col cols="12" sm="6">
                <v-tooltip text="When did this transaction happen or when is it due?" location="top">
                  <template #activator="{ props: tooltipProps }">
                    <v-text-field
                      v-bind="tooltipProps"
                      v-model="form.date"
                      label="Date"
                      type="date"
                      variant="outlined"
                    ></v-text-field>
                  </template>
                </v-tooltip>
              </v-col>

              <v-col cols="12" sm="6">
                <v-tooltip text="A specific classification within this category (e.g., 'Food', 'Transport')" location="top">
                  <template #activator="{ props: tooltipProps }">
                    <v-select
                      v-bind="tooltipProps"
                      v-model="form.subcategory"
                      :items="availableSubcategories"
                      label="Subcategory"
                      variant="outlined"
                      clearable
                    ></v-select>
                  </template>
                </v-tooltip>
              </v-col>

              <v-col cols="12" sm="6">
                <v-tooltip text="How was this transaction paid or received?" location="top">
                  <template #activator="{ props: tooltipProps }">
                    <v-select
                      v-bind="tooltipProps"
                      v-model="form.paymentMethod"
                      :items="availablePaymentMethods"
                      label="Payment Method"
                      variant="outlined"
                      clearable
                    ></v-select>
                  </template>
                </v-tooltip>
              </v-col>

              <v-col cols="12" sm="6">
                <v-tooltip text="Is this recurring or part of an installment plan? (e.g., 'Monthly', '1 of 12')" location="top">
                  <template #activator="{ props: tooltipProps }">
                    <v-text-field
                      v-bind="tooltipProps"
                      v-model="form.installments"
                      label="Installments"
                      variant="outlined"
                    ></v-text-field>
                  </template>
                </v-tooltip>
              </v-col>

              <v-col cols="12" sm="6">
                <v-tooltip text="Add flexible tags (e.g., '#trip', '#fun')" location="top">
                  <template #activator="{ props: tooltipProps }">
                    <v-combobox
                      v-bind="tooltipProps"
                      v-model="form.tags"
                      :items="availableTags"
                      label="Tags"
                      multiple
                      chips
                      variant="outlined"
                    ></v-combobox>
                  </template>
                </v-tooltip>
              </v-col>

              <v-col cols="12">
                <v-tooltip text="Check this if the transaction is already completed. Uncheck it if it is pending." location="top">
                  <template #activator="{ props: tooltipProps }">
                    <v-checkbox
                      v-bind="tooltipProps"
                      v-model="form.isPaid"
                      label="Is Paid?"
                      color="primary"
                      hide-details
                    ></v-checkbox>
                  </template>
                </v-tooltip>
              </v-col>
            </v-row>
          </v-container>
        </v-card-text>

        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="blue-darken-1" variant="text" :disabled="loading" @click="close">
            Cancel
          </v-btn>
          <v-btn color="blue-darken-1" variant="elevated" :loading="loading" @click="save">
            {{ isEditing ? 'Update' : 'Save' }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue';
import {
  GetCategories,
  GetSettings,
  SaveTransaction,
  UpdateTransaction,
} from '../../wailsjs/go/main/App';
import { getCurrencySymbol } from '../utils/currency';

const props = defineProps({
  modelValue: Boolean,
  category: String,
  currencyCode: {
    type: String,
    default: 'USD',
  },
  transaction: {
    type: Object,
    default: null,
  },
});

const emit = defineEmits(['update:modelValue', 'saved']);

const snackbar = reactive({
  show: false,
  text: '',
  color: 'error',
});

const availableCategories = ref([]);
const availableSubcategories = ref([]);
const availablePaymentMethods = ref([]);
const availableTags = ref([]);
const loading = ref(false);

const form = reactive({
  description: '',
  amount: '',
  date: getLocalDate(),
  category: '',
  subcategory: '',
  paymentMethod: '',
  installments: '',
  tags: [],
  isPaid: true,
});

const isEditing = computed(() => Boolean(props.transaction?.id));
const currencySymbol = computed(() => getCurrencySymbol(props.currencyCode));
const amountLabel = computed(() => `Amount (${props.currencyCode})`);

async function loadSettings() {
  try {
    const [categories, subcategories, paymentMethods, tags] = await Promise.all([
      GetCategories(),
      GetSettings('subcategories'),
      GetSettings('payment_methods'),
      GetSettings('tags'),
    ]);
    availableCategories.value = (categories || []).map(item => item.name);
    availableSubcategories.value = (subcategories || []).map(item => item.name);
    availablePaymentMethods.value = (paymentMethods || []).map(item => item.name);
    availableTags.value = (tags || []).map(item => item.name);
  } catch (error) {
    console.error('Error loading transaction settings:', error);
    showError('Could not load the transaction options.');
  }
}

function populateForm() {
  if (!props.transaction) {
    resetForm();
    form.category = props.category || '';
    return;
  }

  form.description = props.transaction.description || '';
  form.amount = props.transaction.amount ?? '';
  form.date = props.transaction.date || getLocalDate();
  form.category = props.transaction.category || props.category || '';
  form.subcategory = props.transaction.subcategory || '';
  form.paymentMethod = props.transaction.payment_method || '';
  form.installments = props.transaction.installments || '';
  form.tags = String(props.transaction.tags || '')
    .split(',')
    .map(tag => tag.trim())
    .filter(Boolean);
  form.isPaid = Boolean(props.transaction.is_paid);
}

function resetForm() {
  form.description = '';
  form.amount = '';
  form.date = getLocalDate();
  form.category = '';
  form.subcategory = '';
  form.paymentMethod = '';
  form.installments = '';
  form.tags = [];
  form.isPaid = true;
}

function getLocalDate() {
  const today = new Date();
  const year = today.getFullYear();
  const month = String(today.getMonth() + 1).padStart(2, '0');
  const day = String(today.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function handleDialogChange(value) {
  if (!value) close();
}

function close() {
  emit('update:modelValue', false);
  resetForm();
}

function showError(message) {
  snackbar.text = message;
  snackbar.color = 'error';
  snackbar.show = true;
}

async function save() {
  const amount = Number(form.amount);
  if (!form.description.trim() || !form.category || !form.date || !Number.isFinite(amount) || amount <= 0) {
    showError('Description, category, date, and a positive amount are required.');
    return;
  }

  loading.value = true;
  const tags = Array.isArray(form.tags) ? form.tags.join(', ') : String(form.tags || '');

  try {
    const transactionArguments = [
      form.description.trim(),
      amount,
      form.date,
      form.category,
      form.subcategory || '',
      form.paymentMethod || '',
      form.installments || '',
      tags,
      form.isPaid,
    ];

    if (isEditing.value) {
      await UpdateTransaction(props.transaction.id, ...transactionArguments);
    } else {
      await SaveTransaction(...transactionArguments);
    }

    emit('saved');
    close();
  } catch (error) {
    console.error('Error saving transaction:', error);
    showError(isEditing.value ? 'Could not update the transaction.' : 'Could not save the transaction.');
  } finally {
    loading.value = false;
  }
}

onMounted(loadSettings);
watch(() => props.modelValue, async value => {
  if (!value) return;
  await loadSettings();
  populateForm();
});
</script>
