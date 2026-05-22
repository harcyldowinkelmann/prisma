<template>
  <v-container fluid class="fill-height align-start">
    <v-row>
      <!-- LEFT SIDEBAR -->
      <v-col cols="12" md="3" class="border-e">
        <v-list nav>
          <v-list-item
            v-for="tab in tabs"
            :key="tab.value"
            :value="tab.value"
            :title="tab.title"
            :active="activeTab === tab.value"
            color="primary"
            @click="activeTab = tab.value"
          ></v-list-item>
        </v-list>
      </v-col>

      <!-- RIGHT CONTENT -->
      <v-col cols="12" md="9">
        <v-card variant="flat">
          <v-card-title class="d-flex align-center pe-2">
            <v-text-field
              v-model="search"
              prepend-inner-icon="mdi-magnify"
              density="compact"
              label="Search..."
              single-line
              flat
              hide-details
              variant="outlined"
            ></v-text-field>
            <v-spacer></v-spacer>
            <v-btn
              color="primary"
              variant="tonal"
              icon="mdi-plus"
              class="ml-4"
              @click="openModal()"
            ></v-btn>
          </v-card-title>

          <v-card-text>
            <v-table>
              <thead>
                <tr>
                  <th class="text-left">Description</th>
                  <th class="text-right" style="width: 120px;">Actions</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in filteredItems" :key="item.uuid">
                  <td>{{ item.name }}</td>
                  <td class="text-right">
                    <v-btn
                      icon="mdi-pencil"
                      variant="text"
                      size="small"
                      color="primary"
                      @click="openModal(item)"
                    ></v-btn>
                    <v-btn
                      icon="mdi-eye-off"
                      variant="text"
                      size="small"
                      color="error"
                      @click="inactivateItem(item.uuid)"
                    ></v-btn>
                  </td>
                </tr>
                <tr v-if="filteredItems.length === 0">
                  <td colspan="2" class="text-center text-disabled py-4">No records found.</td>
                </tr>
              </tbody>
            </v-table>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- ADD/EDIT MODAL -->
    <v-dialog v-model="isModalOpen" max-width="400px">
      <v-card>
        <v-card-title>{{ editingItem ? 'Edit Item' : 'New Item' }}</v-card-title>
        <v-card-text>
          <v-text-field
            v-model="form.name"
            label="Name"
            variant="outlined"
            autofocus
            @keyup.enter="saveItem"
          ></v-text-field>
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="error" variant="text" @click="isModalOpen = false">Cancel</v-btn>
          <v-btn color="primary" variant="flat" @click="saveItem">Save</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue';
import { GetSettings, AddSetting, UpdateSetting, InactivateSetting } from '../../wailsjs/go/main/App';

const tabs = [
  { title: 'Sub Categories', value: 'subcategories' },
  { title: 'Payment Methods', value: 'payment_methods' },
  { title: 'Tags', value: 'tags' }
];

const activeTab = ref('subcategories');
const search = ref('');
const items = ref([]);
const isModalOpen = ref(false);

const editingItem = ref(null);
const form = ref({ name: '' });

const filteredItems = computed(() => {
  if (!search.value) return items.value;
  return items.value.filter(i => i.name.toLowerCase().includes(search.value.toLowerCase()));
});

async function loadData() {
  try {
    const res = await GetSettings(activeTab.value);
    items.value = res || [];
  } catch (err) {
    console.error("Error loading settings:", err);
  }
}

watch(activeTab, () => {
  search.value = '';
  loadData();
});

onMounted(() => {
  loadData();
});

function openModal(item = null) {
  if (item) {
    editingItem.value = item;
    form.value.name = item.name;
  } else {
    editingItem.value = null;
    form.value.name = '';
  }
  isModalOpen.value = true;
}

async function saveItem() {
  if (!form.value.name) return;
  
  try {
    if (editingItem.value) {
      await UpdateSetting(activeTab.value, editingItem.value.uuid, form.value.name);
    } else {
      await AddSetting(activeTab.value, form.value.name);
    }
    isModalOpen.value = false;
    loadData();
  } catch (err) {
    console.error("Error saving item:", err);
  }
}

async function inactivateItem(uuid) {
  if (!confirm("Are you sure you want to inactivate this item? It won't appear in the dropdowns anymore.")) return;
  
  try {
    await InactivateSetting(activeTab.value, uuid);
    loadData();
  } catch (err) {
    console.error("Error inactivating item:", err);
  }
}
</script>
