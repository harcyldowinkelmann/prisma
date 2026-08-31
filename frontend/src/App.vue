<template>
  <v-app>
    <v-main>
      <!-- STATIC HEADER (Always visible) -->
      <Metrics :currency-code="currencyCode" :refresh-key="metricsRefreshKey" />

      <!-- NAVIGATION DIRECTLY BELOW THE HEADER -->
      <v-tabs v-model="activeTab" bg-color="transparent" align-tabs="center" class="mt-2 mb-4">
        <v-tab value="dashboard">Dashboard</v-tab>
        <v-tab value="metrics">Metrics & Reports</v-tab>
        <v-tab value="transactions">All Transactions</v-tab>
        <v-tab value="planning">Planning</v-tab>
        <v-tab value="settings">Settings</v-tab>
      </v-tabs>

      <!-- DYNAMIC CONTENT (Changes with the selected tab) -->
      <v-window v-model="activeTab">
        <!-- TAB 1: DASHBOARD (Current three-column view) -->
        <v-window-item value="dashboard">
          
          <v-container fluid class="pa-4">
            <!-- Header for Dashboard with Add Column Button -->
            <div class="d-flex justify-end mb-4">
              <v-btn color="primary" prepend-icon="mdi-plus" @click="$refs.categoryModalRef.open()">
                Add Column
              </v-btn>
            </div>

            <!-- Horizontal Scroll Container -->
            <div 
              class="d-flex overflow-x-auto flex-nowrap" 
              style="gap: 16px; padding-bottom: 16px;" 
              @wheel="handleScroll"
              ref="scrollContainer"
            >
              <div 
                v-for="cat in categories" 
                :key="cat.uuid" 
                style="min-width: 400px; max-width: 400px;"
                @wheel.stop
              >
                <Body 
                  :title="cat.name" 
                  :items="transactionsByCategory[cat.name] || []"
                  :currency-code="currencyCode"
                  @request-add="openModal" 
                  @request-edit="openEditModal" 
                  @request-inactivate="inactivateTransaction" 
                />
              </div>
            </div>
          </v-container>
        </v-window-item>

        <!-- TAB 2: METRICS & REPORTS -->
        <v-window-item value="metrics">
          <Reports :currency-code="currencyCode" :refresh-key="metricsRefreshKey" />
        </v-window-item>

        <!-- TAB 3: ALL TRANSACTIONS -->
        <v-window-item value="transactions">
          <Transactions
            :currency-code="currencyCode"
            :refresh-key="transactionsRefreshKey"
            @request-add="openModal"
            @request-edit="openEditModal"
            @request-archive="inactivateTransaction"
            @request-restore="restoreTransaction"
            @data-changed="loadAllData"
          />
        </v-window-item>

        <!-- TAB 4: PLANNING -->
        <v-window-item value="planning">
          <Planning
            :currency-code="currencyCode"
            :refresh-key="transactionsRefreshKey"
            @data-changed="loadAllData"
          />
        </v-window-item>

        <!-- TAB 5: SETTINGS -->
        <v-window-item value="settings">
          <Settings
            @currency-changed="onCurrencyChanged"
            @data-restored="onDataRestored"
          />
        </v-window-item>
      </v-window>

      <TransactionModal
        v-model="isModalOpen"
        :category="selectedCategory"
        :currency-code="currencyCode"
        :transaction="editingTransaction"
        @saved="onTransactionSaved"
      ></TransactionModal>

      <CategoryModal ref="categoryModalRef" @saved="loadAllData" />
    </v-main>
  </v-app>
</template>

<script setup>
import Metrics from './components/Metrics.vue'
import Body from './components/Body.vue';
import TransactionModal from './components/TransactionModal.vue';
import CategoryModal from './components/CategoryModal.vue';
import Settings from './components/Settings.vue';
import Transactions from './components/Transactions.vue';
import Reports from './components/Reports.vue';
import Planning from './components/Planning.vue';
import { ref, onMounted } from 'vue';
import { GetTransactions, SoftDeleteTransaction, RestoreTransaction, GetCategories, GetCurrencyCode } from '../wailsjs/go/main/App';

const activeTab = ref('dashboard');
const isModalOpen = ref(false);
const selectedCategory = ref('');
const editingTransaction = ref(null);
const currencyCode = ref('USD');
const metricsRefreshKey = ref(0);
const transactionsRefreshKey = ref(0);

const categories = ref([]);
const transactionsByCategory = ref({});
const scrollContainer = ref(null);

// Horizontal scroll hijacking
function handleScroll(e) {
  if (scrollContainer.value) {
    e.preventDefault();
    scrollContainer.value.scrollLeft += e.deltaY;
  }
}

function openModal(categoryTitle) {
  editingTransaction.value = null;
  selectedCategory.value = categoryTitle || '';
  isModalOpen.value = true;
}

function openEditModal(item) {
  editingTransaction.value = item;
  selectedCategory.value = item.category;
  isModalOpen.value = true;
}

async function inactivateTransaction(uuid) {
  if (!confirm("Are you sure you want to archive this transaction? It will be removed from the totals.")) return;
  
  try {
    await SoftDeleteTransaction(uuid);
    await loadAllData();
  } catch (err) {
    console.error("Failed to archive transaction", err);
    alert("Error archiving transaction: " + err);
  }
}

async function loadAllData() {
  try {
    categories.value = await GetCategories() || [];
    
    // Fetch transactions for each category
    const map = {};
    for (const cat of categories.value) {
      map[cat.name] = await GetTransactions({ category: cat.name }) || [];
    }
    transactionsByCategory.value = map;
    metricsRefreshKey.value += 1;
    transactionsRefreshKey.value += 1;
    
    console.log("Data reloaded from SQLite successfully.");
  } catch (err) {
    console.error("Failed to load transactions", err);
  }
}

async function loadCurrency() {
  try {
    currencyCode.value = await GetCurrencyCode() || 'USD';
  } catch (err) {
    console.error('Failed to load currency settings:', err);
  }
}

async function restoreTransaction(uuid) {
  if (!confirm('Restore this transaction and include it in the dashboard totals?')) return;

  try {
    await RestoreTransaction(uuid);
    await loadAllData();
  } catch (err) {
    console.error('Failed to restore transaction', err);
    alert('Error restoring transaction: ' + err);
  }
}

function onCurrencyChanged(newCurrencyCode) {
  currencyCode.value = newCurrencyCode;
}

async function onDataRestored() {
  await loadCurrency();
  await loadAllData();
}

onMounted(() => {
  loadAllData();
  loadCurrency();
});

// Called when modal finishes saving successfully
function onTransactionSaved() {
  console.log("Transaction saved! Time to reload data...");
  loadAllData();
}
</script>

<style>
  #logo {
    display: block;
    width: 50%;
    height: 50%;
    margin: auto;
    padding: 10% 0 0;
    background-position: center;
    background-repeat: no-repeat;
    background-size: 100% 100%;
    background-origin: content-box;
  }
</style>
