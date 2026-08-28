<template>
  <v-card height="100%" class="d-flex flex-column">
    <v-card-title class="d-flex align-center pa-2">
      <span class="text-subtitle-1 font-weight-bold ml-2">{{ title }}</span>
      <v-spacer></v-spacer>
      <!-- Buttons deferred for future logic -->
      <v-btn icon="mdi-filter" variant="text" size="small" color="grey"></v-btn>
      <v-btn icon="mdi-pencil" variant="text" size="small" color="grey"></v-btn>
      <v-btn icon="mdi-eye-off" variant="text" size="small" color="error"></v-btn>
      
      <v-btn
        icon="mdi-plus-box"
        variant="text"
        size="small"
        color="primary"
        @click="onAddClick"
      ></v-btn>
    </v-card-title>

    <v-divider></v-divider>

    <v-card-text class="pa-0 flex-grow-1 overflow-y-auto" style="min-height: 150px; max-height: 500px;">
      <div v-if="!safeItems.length" class="pa-4 text-caption text-disabled text-center">
        No transactions.
      </div>

      <v-expansion-panels v-else variant="accordion" class="custom-panels" multiple>
        <v-expansion-panel v-for="(item, index) in safeItems" :key="item?.id || index">
          <v-expansion-panel-title>
            <div class="w-100 pe-2">
              <div class="d-flex justify-space-between align-center">
                <div class="text-body-1">{{ item?.description || 'No description' }}</div>
                <div class="text-body-2 font-weight-bold">
                  {{ formatMoney(item?.amount || 0) }}
                </div>
              </div>
              <div class="text-caption text-disabled">{{ formatDate(item?.date) }}</div>
            </div>
          </v-expansion-panel-title>
          
          <v-expansion-panel-text>
            <v-row dense class="text-caption mt-2">
              <v-col cols="6" v-if="item?.subcategory"><strong>Subcategory:</strong> {{ item.subcategory }}</v-col>
              <v-col cols="6" v-if="item?.payment_method"><strong>Payment:</strong> {{ item.payment_method }}</v-col>
              <v-col cols="6" v-if="item?.installments"><strong>Installments:</strong> {{ item.installments }}</v-col>
              <v-col cols="6" v-if="item?.tags"><strong>Tags:</strong> {{ item.tags }}</v-col>
              <v-col cols="6"><strong>Status:</strong> {{ item?.is_paid ? 'Paid' : 'Pending' }}</v-col>
            </v-row>
            <v-divider class="my-2"></v-divider>
            <div class="d-flex justify-end">
              <v-btn size="small" variant="text" color="primary" prepend-icon="mdi-pencil" @click.stop="onEditClick(item)">Edit</v-btn>
              <v-btn size="small" variant="text" color="error" prepend-icon="mdi-eye-off" @click.stop="onInactivateClick(item)">Archive</v-btn>
            </div>
          </v-expansion-panel-text>
        </v-expansion-panel>
      </v-expansion-panels>
    </v-card-text>

    <v-divider></v-divider>

    <v-card-actions class="pa-4 bg-surface-light">
      <strong>Total:</strong>
      <v-spacer></v-spacer>
      <strong>{{ formatMoney(totalValue) }}</strong>
    </v-card-actions>
  </v-card>
</template>

<script setup>
import { computed } from 'vue';
import { formatCurrency } from '../utils/currency';

const props = defineProps({
  title: {
    type: String,
    required: true,
  },
  items: {
    type: Array,
    default: () => [],
  },
  currencyCode: {
    type: String,
    default: 'USD',
  },
});

const emit = defineEmits(['request-add', 'request-edit', 'request-inactivate']);

const safeItems = computed(() => {
  if (!props.items || !Array.isArray(props.items)) return [];
  return props.items.filter(i => i != null);
});

function onAddClick() {
  emit('request-add', props.title)
}

function onEditClick(item) {
  if (item) emit('request-edit', item)
}

function onInactivateClick(item) {
  if (item && item.id) emit('request-inactivate', item.id)
}

const totalValue = computed(() => {
  return safeItems.value.reduce((acc, item) => acc + (item.amount || 0), 0)
});

function formatMoney(value) {
  return formatCurrency(value, props.currencyCode);
}

function formatDate(isoDate) {
  if (!isoDate) return '';
  const parts = isoDate.split('-');
  if (parts.length !== 3) return isoDate;
  return `${parts[1]}/${parts[2]}/${parts[0]}`;
}
</script>

<style scoped>
:deep(.v-expansion-panel-title__icon) {
  align-self: flex-start;
  margin-top: 2px;
}
</style>
