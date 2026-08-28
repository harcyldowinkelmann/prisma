<template>
  <v-container fluid class="fill-height align-start">
    <v-row>
      <!-- LEFT SIDEBAR -->
      <v-col cols="12" md="3" class="border-e">
        <v-list nav>
          <v-list-item
            v-for="tab in tabs"
            :key="tab.value"
            :value="tab.value"
            :title="tab.title"
            :active="activeTab === tab.value"
            color="primary"
            @click="activeTab = tab.value"
          ></v-list-item>
        </v-list>
      </v-col>

      <!-- RIGHT CONTENT -->
      <v-col cols="12" md="9">
        <v-card v-if="activeTab === 'notifications'" variant="flat">
          <v-card-title>Payment Reminders</v-card-title>
          <v-card-subtitle>
            Choose whether Prisma should notify you about due and overdue unpaid expenses.
          </v-card-subtitle>

          <v-card-text class="pt-6">
            <v-list lines="three" class="border rounded-lg">
              <v-list-item
                prepend-icon="mdi-bell-outline"
                title="Enable payment reminders"
                subtitle="The preference is saved locally and applies to the next notification check."
              >
                <template #append>
                  <v-switch
                    :model-value="notificationsEnabled"
                    :loading="notificationsLoading || notificationsSaving"
                    :disabled="notificationsLoading || notificationsSaving"
                    color="primary"
                    hide-details
                    aria-label="Enable payment reminders"
                    @update:model-value="saveNotificationsEnabled"
                  ></v-switch>
                </template>
              </v-list-item>
            </v-list>

            <div class="text-caption text-medium-emphasis mt-3">
              Current status: {{ notificationsEnabled ? 'Enabled' : 'Disabled' }}
            </div>
          </v-card-text>
        </v-card>

        <v-card v-else-if="activeTab === 'currency'" variant="flat">
          <v-card-title>Currency</v-card-title>
          <v-card-subtitle>
            Choose the currency used to display monetary values throughout Prisma.
          </v-card-subtitle>

          <v-card-text class="pt-6">
            <v-select
              v-model="currencyCode"
              :items="currencyOptions"
              item-title="name"
              item-value="code"
              label="Display Currency"
              variant="outlined"
              :loading="currencyLoading || currencySaving"
              :disabled="currencyLoading || currencySaving"
              @update:model-value="saveCurrencyCode"
            ></v-select>

            <div class="text-caption text-medium-emphasis">
              The currency changes formatting only. It does not convert existing transaction values.
            </div>
          </v-card-text>
        </v-card>

        <v-card v-else variant="flat">
          <v-card-title class="d-flex align-center pe-2">
            <v-text-field
              v-model="search"
              prepend-inner-icon="mdi-magnify"
              density="compact"
              label="Search..."
              single-line
              flat
              hide-details
              variant="outlined"
            ></v-text-field>
            <v-spacer></v-spacer>
            <v-btn
              color="primary"
              variant="tonal"
              icon="mdi-plus"
              class="ml-4"
              @click="openModal()"
            ></v-btn>
          </v-card-title>

          <v-card-text>
            <v-table>
              <thead>
                <tr>
                  <th class="text-left">Description</th>
                  <th class="text-right" style="width: 120px;">Actions</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in filteredItems" :key="item.uuid">
                  <td>{{ item.name }}</td>
                  <td class="text-right">
                    <v-btn
                      icon="mdi-pencil"
                      variant="text"
                      size="small"
                      color="primary"
                      @click="openModal(item)"
                    ></v-btn>
                    <v-btn
                      icon="mdi-eye-off"
                      variant="text"
                      size="small"
                      color="error"
                      @click="inactivateItem(item.uuid)"
                    ></v-btn>
                  </td>
                </tr>
                <tr v-if="filteredItems.length === 0">
                  <td colspan="2" class="text-center text-disabled py-4">No records found.</td>
                </tr>
              </tbody>
            </v-table>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- ADD/EDIT MODAL -->
    <v-dialog v-model="isModalOpen" max-width="400px">
      <v-card>
        <v-card-title>{{ editingItem ? 'Edit Item' : 'New Item' }}</v-card-title>
        <v-card-text>
          <v-text-field
            v-model="form.name"
            label="Name"
            variant="outlined"
            autofocus
            @keyup.enter="saveItem"
          ></v-text-field>
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="error" variant="text" @click="isModalOpen = false">Cancel</v-btn>
          <v-btn color="primary" variant="flat" @click="saveItem">Save</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-snackbar
      v-model="feedback.show"
      :color="feedback.color"
      location="top right"
      timeout="3000"
    >
      {{ feedback.message }}
      <template #actions>
        <v-btn variant="text" @click="feedback.show = false">Close</v-btn>
      </template>
    </v-snackbar>
  </v-container>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue';
import {
  GetSettings,
  AddSetting,
  UpdateSetting,
  InactivateSetting,
  GetNotificationsEnabled,
  SetNotificationsEnabled,
  GetCurrencyCode,
  SetCurrencyCode
} from '../../wailsjs/go/main/App';

const emit = defineEmits(['currency-changed']);

const tabs = [
  { title: 'Subcategories', value: 'subcategories' },
  { title: 'Payment Methods', value: 'payment_methods' },
  { title: 'Tags', value: 'tags' },
  { title: 'Currency', value: 'currency' },
  { title: 'Notifications', value: 'notifications' }
];

const activeTab = ref('subcategories');
const search = ref('');
const items = ref([]);
const isModalOpen = ref(false);
const notificationsEnabled = ref(false);
const notificationsLoading = ref(false);
const notificationsSaving = ref(false);
const currencyCode = ref('USD');
const currencyLoading = ref(false);
const currencySaving = ref(false);

const currencyOptions = [
  { code: 'AUD', name: 'Australian Dollar (AUD)' },
  { code: 'BRL', name: 'Brazilian Real (BRL)' },
  { code: 'CAD', name: 'Canadian Dollar (CAD)' },
  { code: 'EUR', name: 'Euro (EUR)' },
  { code: 'GBP', name: 'British Pound (GBP)' },
  { code: 'JPY', name: 'Japanese Yen (JPY)' },
  { code: 'USD', name: 'US Dollar (USD)' },
];

const feedback = reactive({
  show: false,
  message: '',
  color: 'success'
});

const editingItem = ref(null);
const form = ref({ name: '' });

const filteredItems = computed(() => {
  if (!search.value) return items.value;
  return items.value.filter(i => i.name.toLowerCase().includes(search.value.toLowerCase()));
});

async function loadData() {
  if (activeTab.value === 'notifications') {
    notificationsLoading.value = true;
    try {
      notificationsEnabled.value = await GetNotificationsEnabled();
    } catch (err) {
      console.error("Error loading notification settings:", err);
      showFeedback('Could not load the notification preference.', 'error');
    } finally {
      notificationsLoading.value = false;
    }
    return;
  }

  if (activeTab.value === 'currency') {
    currencyLoading.value = true;
    try {
      currencyCode.value = await GetCurrencyCode() || 'USD';
    } catch (err) {
      console.error('Error loading currency settings:', err);
      showFeedback('Could not load the currency preference.', 'error');
    } finally {
      currencyLoading.value = false;
    }
    return;
  }

  try {
    const res = await GetSettings(activeTab.value);
    items.value = res || [];
  } catch (err) {
    console.error("Error loading settings:", err);
  }
}

watch(activeTab, () => {
  search.value = '';
  isModalOpen.value = false;
  loadData();
});

onMounted(() => {
  loadData();
});

function openModal(item = null) {
  if (item) {
    editingItem.value = item;
    form.value.name = item.name;
  } else {
    editingItem.value = null;
    form.value.name = '';
  }
  isModalOpen.value = true;
}

function showFeedback(message, color) {
  feedback.message = message;
  feedback.color = color;
  feedback.show = true;
}

async function saveNotificationsEnabled(enabled) {
  if (notificationsSaving.value) return;

  notificationsSaving.value = true;
  try {
    await SetNotificationsEnabled(Boolean(enabled));
    notificationsEnabled.value = Boolean(enabled);
    showFeedback(
      enabled ? 'Payment reminders enabled.' : 'Payment reminders disabled.',
      'success'
    );
  } catch (err) {
    console.error("Error saving notification settings:", err);
    showFeedback('Could not save the notification preference.', 'error');
  } finally {
    notificationsSaving.value = false;
  }
}

async function saveCurrencyCode(newCurrencyCode) {
  if (currencySaving.value || !newCurrencyCode) return;

  currencySaving.value = true;
  try {
    await SetCurrencyCode(newCurrencyCode);
    currencyCode.value = newCurrencyCode;
    emit('currency-changed', newCurrencyCode);
    showFeedback(`Display currency changed to ${newCurrencyCode}.`, 'success');
  } catch (err) {
    console.error('Error saving currency settings:', err);
    showFeedback('Could not save the currency preference.', 'error');
    await loadData();
  } finally {
    currencySaving.value = false;
  }
}

async function saveItem() {
  if (!form.value.name) return;
  
  try {
    if (editingItem.value) {
      await UpdateSetting(activeTab.value, editingItem.value.uuid, form.value.name);
    } else {
      await AddSetting(activeTab.value, form.value.name);
    }
    isModalOpen.value = false;
    loadData();
  } catch (err) {
    console.error("Error saving item:", err);
  }
}

async function inactivateItem(uuid) {
  if (!confirm("Are you sure you want to inactivate this item? It won't appear in the dropdowns anymore.")) return;
  
  try {
    await InactivateSetting(activeTab.value, uuid);
    loadData();
  } catch (err) {
    console.error("Error inactivating item:", err);
  }
}
</script>
