<template>
    <v-container fluid>
        <v-sheet class="ma-2 pa-4 border-sm" rounded="lg">
            <v-row no-gutters>
                <v-col cols="12">
                    <div class="mb-4">
                        <h2>Métricas de <strong>Novembro</strong></h2>
                    </div>
                    <v-row dense>
                        <v-col
                            v-for="metric in metrics"
                            :key="metric.title"
                            cols="12"
                            sm="4"
                            md="2"
                        >
                            <v-card
                                class="pa-2 fill-height rounded-lg"
                                :color="getMetricColor(metric.type)"
                                variant="tonal"
                            >
                                <v-card-title class="text-caption pa-0 mb-1">
                                    {{ metric.title }}
                                </v-card-title>
            
                                <v-card-text class="text-h6 pa-0 font-weight-bold">
                                    {{ metric.value }}
                                </v-card-text>
                            </v-card>
                        </v-col>
                    </v-row>
                </v-col>
            </v-row>
        </v-sheet>
    </v-container>
</template>

<script setup>
    import { ref } from 'vue';

    // Dados estáticos (mock) para popular o dashboard.
    const metrics = ref([
        { title: 'Receita', value: 'R$ 3.000,00', type: 'receita' },
        { title: 'Despesas Fixas', value: 'R$ 2.115,00', type: 'despesa' },
        { title: 'Despesas Variáveis', value: 'R$ 525,35', type: 'despesa' },
        { title: 'Total de gasto', value: 'R$ 2.640,35', type: 'despesa_total' },
        { title: '% de gasto', value: '88%', type: 'neutro' },
        { title: '% de sobra', value: '12%', type: 'neutro' },
    ]);

    /**
     * Retorna uma cor baseada no tipo de métrica.
     */
    function getMetricColor(type) {
        switch (type) {
            case 'receita':
            return 'success'; // Verde
            case 'despesa':
            return 'warning'; // Laranja/Amarelo
            case 'despesa_total':
            return 'error'; // Vermelho
            default:
            return 'surface-variant'; // Um cinza padrão do Vuetify
        }
    }
</script>

<style scoped>
    .text-caption {
        line-height: 1.25rem; /* Garante que títulos longos não fiquem estranhos */
    }
</style>
