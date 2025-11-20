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

    <v-card-text>
      <div v-if="items.length === 0" class="text-caption text-disabled">
        Nenhum lançamento.
      </div>
    </v-card-text>

    <v-divider></v-divider>
    <v-card-actions class="pa-4">
      <strong>Total:</strong>
      <v-spacer></v-spacer>
      <strong>R$ 0,00</strong>
    </v-card-actions>
  </v-card>
</template>

<script setup>
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
</script>
