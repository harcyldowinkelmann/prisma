<template>
    <v-container fluid>
        <v-sheet class="ma-2 pa-4 border-sm" rounded="lg">
            <v-row no-gutters>
                <v-col cols="12">
                    <div class="mb-4">
                        <h2><strong>November</strong> Metrics</h2>
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

    // Static data (mock) to populate the dashboard.
    const metrics = ref([
        { title: 'Income', value: '$3,000.00', type: 'income' },
        { title: 'Fixed Expenses', value: '$2,115.00', type: 'expense' },
        { title: 'Variable Expenses', value: '$525.35', type: 'expense' },
        { title: 'Total spent', value: '$2,640.35', type: 'expense_total' },
        { title: '% spent', value: '88%', type: 'neutral' },
        { title: '% remaining', value: '12%', type: 'neutral' },
    ]);

    /**
     * Returns a color based on the metric type.
     */
    function getMetricColor(type) {
        switch (type) {
            case 'income':
            return 'success'; // Green
            case 'expense':
            return 'warning'; // Orange/Yellow
            case 'expense_total':
            return 'error'; // Red
            default:
            return 'surface-variant'; // Vuetify default grey
        }
    }
</script>

<style scoped>
    .text-caption {
        line-height: 1.25rem; /* Ensures long titles don't look weird */
    }
</style>
