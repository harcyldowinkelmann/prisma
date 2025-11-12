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
                <Body title="Receitas" />
            </v-col>

            <!-- COLUNA 2: DESPESAS FIXAS -->
            <v-col cols="12" md="4">
                <Body title="Despesas Fixas" />
            </v-col>

            <!-- COLUNA 3: DESPESAS VARIÁVEIS -->
            <v-col cols="12" md="4">
                <Body title="Despesas Variáveis" />
            </v-col>
          </v-row>
        </v-container>
    </v-main>
  </v-app>
</template>

<script setup>
import Metricas from './components/Metricas.vue'
import Body from './components/Body.vue';
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
