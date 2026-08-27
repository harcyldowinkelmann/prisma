<template>
  <v-app>
    <v-main>
      <!-- STATIC HEADER (Always visible) -->
      <Metrics />

      <!-- NAVIGATION DIRECTLY BELOW THE HEADER -->
      <v-tabs v-model="activeTab" bg-color="transparent" align-tabs="center" class="mt-2 mb-4">
        <v-tab value="dashboard">Dashboard</v-tab>
        <v-tab value="metrics">Metrics & Reports</v-tab>
        <v-tab value="transactions">All Transactions</v-tab>
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
                  @request-add="openModal" 
                  @request-edit="openEditModal" 
                  @request-inactivate="inactivateTransaction" 
                />
              </div>
            </div>
          </v-container>
        </v-window-item>

        <!-- TAB 2: METRICS & REPORTS (Placeholder) -->
        <v-window-item value="metrics">
          <v-container class="text-center mt-10">
            <h2 class="text-disabled">Metrics & Reports Module Coming Soon...</h2>
          </v-container>
        </v-window-item>

        <!-- TAB 3: ALL TRANSACTIONS (Placeholder) -->
        <v-window-item value="transactions">
          <v-container class="text-center mt-10">
            <h2 class="text-disabled">Full Transactions List Coming Soon...</h2>
          </v-container>
        </v-window-item>

        <!-- TAB 4: SETTINGS -->
        <v-window-item value="settings">
          <Settings />
        </v-window-item>
      </v-window>

      <TransactionModal
        v-model="isModalOpen"
        :category="selectedCategory"
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
import { ref, onMounted } from 'vue';
import { GetTransactions, SoftDeleteTransaction, GetCategories } from '../wailsjs/go/main/App';

const activeTab = ref('dashboard');
const isModalOpen = ref(false);
const selectedCategory = ref('');

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
  selectedCategory.value = categoryTitle;
  isModalOpen.value = true;
}

function openEditModal(item) {
  // TODO: Create or modify TransactionModal to accept an item to edit
  alert("Transaction editing will be implemented in the next phase!");
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
    
    console.log("Data reloaded from SQLite successfully.");
  } catch (err) {
    console.error("Failed to load transactions", err);
  }
}

onMounted(() => {
  loadAllData();
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
