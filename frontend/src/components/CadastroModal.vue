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
                <v-btn variant="text" @click="snackbar.show = false">Fechar</v-btn>
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
                    Nova Transação: <span class="text-primary">{{ categoria }}</span>
                </v-card-title>

                <v-card-text>
                    <v-container>
                        <v-row>
                            <v-col cols="12">
                                <v-text-field
                                    v-model="form.descricao"
                                    label="Descrição"
                                    required
                                    variant="outlined"
                                    autofocus
                                ></v-text-field>
                            </v-col>
                            
                            <v-col cols="6">
                                <v-text-field
                                    v-model="form.valor"
                                    label="Valor (R$)"
                                    type="number"
                                    prefix="R$"
                                    variant="outlined"
                                ></v-text-field>
                            </v-col>

                            <v-col cols="6">
                                <v-text-field
                                    v-model="form.data"
                                    label="Data"
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
                        Cancelar
                    </v-btn>

                    <v-btn 
                        color="blue-darken-1" 
                        variant="elevated" 
                        @click="save"
                        :loading="loading"
                    >
                        Salvar
                    </v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
  </div>
</template>

<script setup>
    import { ref, reactive } from 'vue';
    import { SalvarLancamento } from '../../wailsjs/go/main/App';

    const snackbar = reactive({
        show: false,
        text: '',
        color: 'error'
    });

    // Props: O modal precisa saber se está aberto e qual a categoria
    const props = defineProps({
        modelValue: Boolean, // Controla visibilidade (v-model)
        categoria: String,
    });

    // Emits: Para fechar o modal ou avisar que salvou
    const emit = defineEmits(['update:modelValue', 'saved']);

    // Estado local do formulário
    const loading = ref(false);
    const form = reactive({
        descricao: '',
        valor: '',
        data: new Date().toISOString().substr(0, 10), // Data de hoje YYYY-MM-DD
    });

    // Função para fechar o modal
    function close() {
        emit('update:modelValue', false);
        form.descricao = '';
        form.valor = '';
    }

    function save() {
        if (!form.descricao || !form.valor) return;

        loading.value = true;

        const valorFloat = parseFloat(form.valor);

        // Chama o Backend GO
        SalvarLancamento(form.descricao, valorFloat, form.data, props.categoria)
            .then((msg) => {
                console.log(msg);
                emit('saved'); // Avisa o App.vue que salvou
                close();
            })
            .catch((err) => {
                console.error("Erro ao salvar:", err);
                snackbar.text = "Não foi possível salvar o lançamento!";
                snackbar.color = 'error';
                snackbar.show = true;
            })
            .finally(() => {
                loading.value = false;
            });
    }
</script>
