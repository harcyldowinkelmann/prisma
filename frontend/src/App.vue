<template>
  <v-app>
    <v-main>
      <Metrics />

      <v-divider />
      
      <v-container fluid class="pa-4">
          <!-- This v-row holds the 3 columns -->
          <v-row dense>
          
            <!-- Incomes Column -->
            <v-col cols="12" md="4">
                <Body title="Incomes" :items="incomes" @request-add="openModal" />
            </v-col>

            <!-- COLUMN 2: FIXED EXPENSES -->
            <v-col cols="12" md="4">
                <Body title="Fixed Expenses" :items="fixedExpenses" @request-add="openModal" />
            </v-col>

            <!-- COLUMN 3: VARIABLE EXPENSES -->
            <v-col cols="12" md="4">
                <Body title="Variable Expenses" :items="variableExpenses" @request-add="openModal" />
            </v-col>
          </v-row>
      </v-container>

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
import { ref, onMounted } from 'vue';
import { GetTransactions } from '../wailsjs/go/main/App';

const isModalOpen = ref(false);
const selectedCategory = ref('');

const incomes = ref([]);
const fixedExpenses = ref([]);
const variableExpenses = ref([]);

function openModal(categoryTitle) {
  selectedCategory.value = categoryTitle;
  isModalOpen.value = true;
}

async function loadAllData() {
  try {
    incomes.value = await GetTransactions({ category: "Incomes" });

    fixedExpenses.value = await GetTransactions({ category: "Fixed Expenses" });

    variableExpenses.value = await GetTransactions({ category: "Variable Expenses" });
    
    console.log("Data reloaded from SQLite successfully.");
  } catch (err) {
    console.error("Error loading data:", err);
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
