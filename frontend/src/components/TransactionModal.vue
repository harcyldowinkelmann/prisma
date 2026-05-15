<template>
  <!-- Snackbar de aviso -->
  <div>
        <v-snackbar
            v-model="snackbar.show"
            :color="snackbar.color"
            timeout="3000"
            location="top right"
        >
            {{ snackbar.text }}
            <template v-slot:actions>
                <v-btn variant="text" @click="snackbar.show = false">Close</v-btn>
            </template>
        </v-snackbar>

        <!-- v-dialog cria a janela flutuante -->
        <v-dialog
            :model-value="modelValue"
            @update:model-value="close"
            max-width="500px"
        >
            <v-card>
                <v-card-title>
                    New Transaction: <span class="text-primary">{{ category }}</span>
                </v-card-title>

                <v-card-text>
                    <v-container>
                        <v-row>
                            <v-col cols="12">
                                <v-text-field
                                    v-model="form.description"
                                    label="Description"
                                    required
                                    variant="outlined"
                                    autofocus
                                ></v-text-field>
                            </v-col>
                            
                            <v-col cols="6">
                                <v-text-field
                                    v-model="form.amount"
                                    label="Amount ($)"
                                    type="number"
                                    prefix="$"
                                    variant="outlined"
                                ></v-text-field>
                            </v-col>

                            <v-col cols="6">
                                <v-text-field
                                    v-model="form.date"
                                    label="Date"
                                    type="date"
                                    variant="outlined"
                                ></v-text-field>
                            </v-col>
                        </v-row>
                    </v-container>
                </v-card-text>

                <v-card-actions>

                    <v-spacer></v-spacer>

                    <v-btn color="blue-darken-1" variant="text" @click="close">
                        Cancel
                    </v-btn>

                    <v-btn 
                        color="blue-darken-1" 
                        variant="elevated" 
                        @click="save"
                        :loading="loading"
                    >
                        Save
                    </v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
  </div>
</template>

<script setup>
    import { ref, reactive } from 'vue';
    import { SaveTransaction } from '../../wailsjs/go/main/App';

    const snackbar = reactive({
        show: false,
        text: '',
        color: 'error'
    });

    // Props: The modal needs to know if it's open and which category it is
    const props = defineProps({
        modelValue: Boolean, // Controls visibility (v-model)
        category: String,
    });

    // Emits: To close the modal or notify that it saved
    const emit = defineEmits(['update:modelValue', 'saved']);

    // Local form state
    const loading = ref(false);
    const form = reactive({
        description: '',
        amount: '',
        date: new Date().toISOString().substr(0, 10), // Today's date YYYY-MM-DD
    });

    // Function to close the modal
    function close() {
        emit('update:modelValue', false);
        form.description = '';
        form.amount = '';
    }

    function save() {
        if (!form.description || !form.amount) return;

        loading.value = true;

        const amountFloat = parseFloat(form.amount);

        // Call GO Backend
        SaveTransaction(form.description, amountFloat, form.date, props.category)
            .then((msg) => {
                console.log(msg);
                emit('saved'); // Notify App.vue that it saved
                close();
            })
            .catch((err) => {
                console.error("Error saving:", err);
                snackbar.text = "Could not save transaction!";
                snackbar.color = 'error';
                snackbar.show = true;
            })
            .finally(() => {
                loading.value = false;
            });
    }
</script>
