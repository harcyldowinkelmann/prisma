<template>
  <v-app>
    <v-main>
      <!-- HEADER ESTÁTICO (Nunca some) -->
      <Metrics />

      <!-- MENU COLADO LOGO ABAIXO DO HEADER -->
      <v-tabs v-model="activeTab" bg-color="transparent" align-tabs="center" class="mt-2 mb-4">
        <v-tab value="dashboard">Dashboard</v-tab>
        <v-tab value="metrics">Metrics & Reports</v-tab>
        <v-tab value="transactions">All Transactions</v-tab>
        <v-tab value="settings">Settings</v-tab>
      </v-tabs>

      <!-- CONTEÚDO DINÂMICO (Muda conforme o menu) -->
      <v-window v-model="activeTab">
        <!-- TAB 1: DASHBOARD (Current View - 3 Colunas) -->
        <v-window-item value="dashboard">
          
          <v-container fluid class="pa-4">
              <!-- This v-row holds the 3 columns -->
              <v-row class="ma-0">
          <!-- TAB 1: DASHBOARD / COLUMNS -->
          <v-col cols="12" md="4" class="pa-2">
            <Body title="Incomes" :items="incomes" @request-add="openModal" @request-edit="openEditModal" @request-inactivate="inactivateTransaction" />
          </v-col>

          <v-col cols="12" md="4" class="pa-2">
            <Body title="Fixed Expenses" :items="fixedExpenses" @request-add="openModal" @request-edit="openEditModal" @request-inactivate="inactivateTransaction" />
          </v-col>

          <v-col cols="12" md="4" class="pa-2">
            <Body title="Variable Expenses" :items="variableExpenses" @request-add="openModal" @request-edit="openEditModal" @request-inactivate="inactivateTransaction" />
          </v-col>
        </v-row>
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
    </v-main>
  </v-app>
</template>

<script setup>
import Metrics from './components/Metrics.vue'
import Body from './components/Body.vue';
import TransactionModal from './components/TransactionModal.vue';
import Settings from './components/Settings.vue';
import { ref, onMounted } from 'vue';
import { GetTransactions, SoftDeleteTransaction } from '../wailsjs/go/main/App';

const activeTab = ref('dashboard');
const isModalOpen = ref(false);
const selectedCategory = ref('');

const incomes = ref([]);
const fixedExpenses = ref([]);
const variableExpenses = ref([]);

function openModal(categoryTitle) {
  selectedCategory.value = categoryTitle;
  isModalOpen.value = true;
}

function openEditModal(item) {
  // TODO: Create or modify TransactionModal to accept an item to edit
  alert("Edição de transação será implementada na próxima fase!");
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
    incomes.value = await GetTransactions({ category: "Incomes" }) || [];
    fixedExpenses.value = await GetTransactions({ category: "Fixed Expenses" }) || [];
    variableExpenses.value = await GetTransactions({ category: "Variable Expenses" }) || [];
    
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
