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

        <!-- v-dialog creates the modal window -->
        <v-dialog
            :model-value="modelValue"
            @update:model-value="close"
            max-width="600px"
        >
            <v-card>
                <v-card-title>
                    New Transaction: <span class="text-primary">{{ category }}</span>
                </v-card-title>

                <v-card-text>
                    <v-container>
                        <v-row>
                            <v-col cols="12">
                                <v-tooltip text="Briefly describe what this transaction is about (e.g., 'Walmart Groceries')" location="top">
                                    <template v-slot:activator="{ props }">
                                        <v-text-field v-bind="props" v-model="form.description" label="Description" required variant="outlined" autofocus></v-text-field>
                                    </template>
                                </v-tooltip>
                            </v-col>
                            
                            <v-col cols="6">
                                <v-tooltip text="The total value of the transaction" location="top">
                                    <template v-slot:activator="{ props }">
                                        <v-text-field v-bind="props" v-model="form.amount" label="Amount ($)" type="number" prefix="$" variant="outlined"></v-text-field>
                                    </template>
                                </v-tooltip>
                            </v-col>

                            <v-col cols="6">
                                <v-tooltip text="When did this transaction happen or when is it due?" location="top">
                                    <template v-slot:activator="{ props }">
                                        <v-text-field v-bind="props" v-model="form.date" label="Date" type="date" variant="outlined"></v-text-field>
                                    </template>
                                </v-tooltip>
                            </v-col>

                            <v-col cols="6">
                                <v-tooltip text="A specific classification within this column (e.g., 'Food', 'Transport')" location="top">
                                    <template v-slot:activator="{ props }">
                                        <v-select v-bind="props" v-model="form.subcategory" :items="availableSubcategories" label="Subcategory" variant="outlined"></v-select>
                                    </template>
                                </v-tooltip>
                            </v-col>

                            <v-col cols="6">
                                <v-tooltip text="Where did the money come from? (e.g., 'Credit Card', 'Cash', 'Bank Transfer')" location="top">
                                    <template v-slot:activator="{ props }">
                                        <v-select v-bind="props" v-model="form.paymentMethod" :items="availablePaymentMethods" label="Payment Method" variant="outlined"></v-select>
                                    </template>
                                </v-tooltip>
                            </v-col>

                            <v-col cols="6">
                                <v-tooltip text="Is this a recurring bill or an installment? (e.g., 'Monthly', '1 of 12')" location="top">
                                    <template v-slot:activator="{ props }">
                                        <v-text-field v-bind="props" v-model="form.installments" label="Installments" variant="outlined"></v-text-field>
                                    </template>
                                </v-tooltip>
                            </v-col>

                            <v-col cols="6">
                                <v-tooltip text="Add flexible tags (e.g., '#trip, #fun')" location="top">
                                    <template v-slot:activator="{ props }">
                                        <v-combobox v-bind="props" v-model="form.tags" :items="availableTags" label="Tags" multiple chips variant="outlined"></v-combobox>
                                    </template>
                                </v-tooltip>
                            </v-col>

                            <v-col cols="12">
                                <v-tooltip text="Check this if the transaction is already completed. Uncheck if it's pending." location="top">
                                    <template v-slot:activator="{ props }">
                                        <v-checkbox v-bind="props" v-model="form.isPaid" label="Is Paid?" color="primary" hide-details></v-checkbox>
                                    </template>
                                </v-tooltip>
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
    import { ref, reactive, onMounted, watch } from 'vue';
    import { SaveTransaction, GetSettings } from '../../wailsjs/go/main/App';

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

    const availableSubcategories = ref([]);
    const availablePaymentMethods = ref([]);
    const availableTags = ref([]);

    async function loadSettings() {
        try {
            availableSubcategories.value = (await GetSettings('subcategories'))?.map(i => i.name) || [];
            availablePaymentMethods.value = (await GetSettings('payment_methods'))?.map(i => i.name) || [];
            availableTags.value = (await GetSettings('tags'))?.map(i => i.name) || [];
        } catch (err) {
            console.error("Error loading settings in modal", err);
        }
    }

    onMounted(() => {
        loadSettings();
    });

    watch(() => props.modelValue, (newVal) => {
        if (newVal) loadSettings();
    });

    // Local form state
    const loading = ref(false);
    const form = reactive({
        description: '',
        amount: '',
        date: new Date().toISOString().substr(0, 10), // Today's date YYYY-MM-DD
        subcategory: '',
        paymentMethod: '',
        installments: '',
        tags: [],
        isPaid: true
    });

    // Function to close the modal
    function close() {
        emit('update:modelValue', false);
        form.description = '';
        form.amount = '';
        form.subcategory = '';
        form.paymentMethod = '';
        form.installments = '';
        form.tags = [];
        form.isPaid = true;
    }

    function save() {
        if (!form.description || !form.amount) return;

        loading.value = true;

        const amountFloat = parseFloat(form.amount);
        const tagsString = form.tags.join(', ');

        // Call GO Backend
        SaveTransaction(form.description, amountFloat, form.date, props.category, form.subcategory, form.paymentMethod, form.installments, tagsString, form.isPaid)
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
