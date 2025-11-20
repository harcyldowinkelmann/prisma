<template>
  <v-app>
    <v-main>
      <Metricas />

      <v-divider />
      
      <v-container fluid class="pa-4">
          <!-- Esta v-row segura as 3 colunas -->
          <v-row dense>
          
            <!-- Coluna de 'Receitas' -->
            <v-col cols="12" md="4">
                <Body title="Receitas" :items="receitas" @request-add="openModal" />
            </v-col>

            <!-- COLUNA 2: DESPESAS FIXAS -->
            <v-col cols="12" md="4">
                <Body title="Despesas Fixas" :items="despesasFixas" @request-add="openModal" />
            </v-col>

            <!-- COLUNA 3: DESPESAS VARIÁVEIS -->
            <v-col cols="12" md="4">
                <Body title="Despesas Variáveis" :items="despesasVariaveis" @request-add="openModal" />
            </v-col>
          </v-row>
      </v-container>

      <CadastroModal
        v-model="isModalOpen"
        :categoria="selectedCategory"
        @saved="onTransactionSaved"
      ></CadastroModal>
    </v-main>
  </v-app>
</template>

<script setup>
import Metricas from './components/Metricas.vue'
import Body from './components/Body.vue';
import CadastroModal from './components/CadastroModal.vue';
import { ref, onMounted } from 'vue';
import { BuscarLancamentos } from '../wailsjs/go/main/App';

const isModalOpen = ref(false);
const selectedCategory = ref('');

const receitas = ref([]);
const despesasFixas = ref([]);
const despesasVariaveis = ref([]);

function openModal(categoryTitle) {
  selectedCategory.value = categoryTitle;
  isModalOpen.value = true;
}

async function loadAllData() {
  try {
    receitas.value = await BuscarLancamentos({ categoria: "Receitas" });

    despesasFixas.value = await BuscarLancamentos({ categoria: "Despesas Fixas" });

    despesasVariaveis.value = await BuscarLancamentos({ categoria: "Despesas Variáveis" });
    
    console.log("Dados recarregados do SQLite com sucesso.");
  } catch (err) {
    console.error("Erro ao carregar dados:", err);
  }
}

onMounted(() => {
  loadAllData();
});

// Chama quando o modal termina de salvar com sucesso
function onTransactionSaved() {
  console.log("Transação salva! Hora de recarregar os dados...");
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
