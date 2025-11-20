<template>
  <v-card height="100%">
    <v-card-title class="d-flex align-center">
      <span>{{ title }}</span>
      <v-spacer></v-spacer>
      <v-btn
        icon="mdi-plus-box"
        variant="text"
        size="small"
        @click="onAddClick"
      ></v-btn>
    </v-card-title>

    <v-divider></v-divider>

    <v-card-text class="pa-0 overflow-y-auto" style="max-height: 400px;">

      <div v-if="items.length === 0" class="pa-4 text-caption text-disabled text-center">
        Nenhum lançamento.
      </div>

      <v-list v-else lines="two" density="compact">
        <v-list-item
          v-for="item in items"
          :key="item.id"
          :title="item.descricao"
          :subtitle="formatData(item.data)"
        >
          <template v-slot:append>
            <div
              class="text-body-2 font-weight-bold"
            >
              {{  formatCurrency(item.valor)  }}
            </div>
          </template>
        </v-list-item>
      </v-list>

    </v-card-text>

    <v-divider></v-divider>

    <v-card-actions class="pa-4 bg-surface-light">
      <strong>Total:</strong>
      <v-spacer></v-spacer>
      <strong>{{ formatCurrency(totalValue) }}</strong>
    </v-card-actions>
  </v-card>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
  title: {
    type: String,
    required: true,
  },
  items: {
    type: Array,
    default: () => [],
  },
});

const emit = defineEmits(['request-add']);

function onAddClick() {
  emit('request-add', props.title)
}

const totalValue = computed(() => {
  return props.items.reduce((acc, item) => acc + item.valor, 0)
});

function formatCurrency(value) {
  return new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency: 'BRL',
  }).format(value);
}

function formatData(isoDate) {
  if (!isoDate) return '';
  const [year, month, day] = isoDate.split('-');
  return `${day}/${month}/${year}`;
}
</script>
