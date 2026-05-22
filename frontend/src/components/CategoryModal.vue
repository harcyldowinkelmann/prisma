<template>
  <v-dialog v-model="isOpen" max-width="500px">
    <v-card class="bg-surface">
      <v-card-title class="text-h5 font-weight-bold pa-4 pb-2">
        Add New Column
      </v-card-title>
      <v-card-text class="pa-4">
        <v-form ref="formRef" @submit.prevent="save">
          <v-text-field
            v-model="form.name"
            label="Column Name"
            variant="outlined"
            density="comfortable"
            class="mb-4"
            :rules="[v => !!v || 'Name is required']"
          ></v-text-field>

          <div class="text-subtitle-2 mb-2">Column Type (Math Nature)</div>
          <v-radio-group v-model="form.type" inline :rules="[v => !!v || 'Type is required']">
            <v-radio label="Income (Adds to balance)" :value="1" color="success"></v-radio>
            <v-radio label="Expense (Subtracts from balance)" :value="-1" color="error"></v-radio>
          </v-radio-group>
        </v-form>
      </v-card-text>
      <v-card-actions class="pa-4 pt-0">
        <v-spacer></v-spacer>
        <v-btn color="error" variant="text" @click="close">Cancel</v-btn>
        <v-btn color="primary" variant="elevated" @click="save">Save Column</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup>
import { ref, reactive } from 'vue';
import { AddCategory } from '../../wailsjs/go/main/App';

const emit = defineEmits(['saved']);
const isOpen = ref(false);
const formRef = ref(null);

const form = reactive({
  name: '',
  type: -1
});

const open = () => {
  form.name = '';
  form.type = -1; // Default to Expense
  isOpen.value = true;
};

const close = () => {
  isOpen.value = false;
};

const save = async () => {
  const { valid } = await formRef.value.validate();
  if (!valid) return;

  try {
    await AddCategory(form.name, form.type);
    emit('saved');
    close();
  } catch (error) {
    console.error("Error saving category:", error);
    alert("Failed to save column: " + error);
  }
};

defineExpose({ open });
</script>
