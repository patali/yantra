<template>
  <v-container
    fluid
    class="pa-6"
  >
    <v-row>
      <v-col cols="12">
        <div class="d-flex justify-space-between align-center mb-4">
          <h1 class="text-h4">
            Users
          </h1>
          <v-btn
            color="primary"
            prepend-icon="mdi-plus"
            @click="createDialog = true"
          >
            Create User
          </v-btn>
        </div>
      </v-col>
    </v-row>

    <v-row>
      <v-col cols="12">
        <v-card elevation="2">
          <v-card-text>
            <v-data-table
              :headers="headers"
              :items="admins"
              :loading="loading"
              items-per-page="10"
            >
              <template #[`item.createdAt`]="{ item }">
                {{ new Date(item.createdAt).toLocaleString() }}
              </template>

              <template #[`item.actions`]="{ item }">
                <v-btn
                  v-if="item.id !== currentAdminId"
                  icon="mdi-delete"
                  size="small"
                  variant="text"
                  color="error"
                  @click="confirmDelete(item)"
                />
                <v-chip
                  v-else
                  size="small"
                  color="primary"
                >
                  You
                </v-chip>
              </template>
            </v-data-table>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Create User Dialog -->
    <v-dialog
      v-model="createDialog"
      max-width="500"
    >
      <v-card>
        <v-card-title>Create New User</v-card-title>
        <v-card-text>
          <v-form ref="adminForm">
            <v-text-field
              v-model="newAdmin.username"
              label="Username"
              variant="outlined"
              :rules="[rules.required, rules.minLength(3)]"
              class="mb-4"
            />

            <v-text-field
              v-model="newAdmin.email"
              label="Email"
              type="email"
              variant="outlined"
              :rules="[rules.required, rules.email]"
              class="mb-4"
            />

            <v-text-field
              v-model="newAdmin.password"
              label="Password"
              type="password"
              variant="outlined"
              :rules="[rules.required, rules.minLength(8)]"
            />
          </v-form>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="createDialog = false">
            Cancel
          </v-btn>
          <v-btn
            color="primary"
            :loading="creating"
            @click="createAdmin"
          >
            Create
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete Confirmation Dialog -->
    <v-dialog
      v-model="deleteDialog"
      max-width="400"
    >
      <v-card>
        <v-card-title>Delete User</v-card-title>
        <v-card-text>
          Are you sure you want to delete user "{{ adminToDelete?.username }}"?
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="deleteDialog = false">
            Cancel
          </v-btn>
          <v-btn
            color="error"
            :loading="deleting"
            @click="deleteAdmin"
          >
            Delete
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-snackbar
      v-model="snackbar"
      :color="snackbarColor"
      timeout="3000"
    >
      {{ snackbarText }}
    </v-snackbar>
  </v-container>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive, computed } from "vue";
import api from "@/services/api";
import { useAuthStore } from "@/stores/auth";
import type { Admin } from "@/types";

const authStore = useAuthStore();
const admins = ref<Admin[]>([]);
const loading = ref(false);
const createDialog = ref(false);
const deleteDialog = ref(false);
const creating = ref(false);
const deleting = ref(false);
const adminToDelete = ref<Admin | null>(null);

const snackbar = ref(false);
const snackbarText = ref("");
const snackbarColor = ref("success");

const currentAdminId = computed(() => authStore.admin?.id);

const newAdmin = reactive({
  username: "",
  email: "",
  password: "",
});

const headers = [
  { title: "Username", value: "username" },
  { title: "Email", value: "email" },
  { title: "Created At", value: "createdAt" },
  { title: "Actions", value: "actions", sortable: false },
];

const rules = {
  required: (v: string) => !!v || "Required",
  minLength: (min: number) => (v: string) =>
    v.length >= min || `Minimum ${min} characters`,
  email: (v: string) =>
    /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v) || "Invalid email",
};

const fetchAdmins = async () => {
  try {
    loading.value = true;
    const response = await api.get("/users");
    admins.value = response.data;
  } catch (_error) {
    showSnackbar("Failed to fetch users", "error");
  } finally {
    loading.value = false;
  }
};

const createAdmin = async () => {
  try {
    creating.value = true;
    await api.post("/users", newAdmin);
    showSnackbar("User created successfully", "success");
    createDialog.value = false;
    resetForm();
    fetchAdmins();
  } catch (error: any) {
    showSnackbar(
      error.response?.data?.error || "Failed to create user",
      "error"
    );
  } finally {
    creating.value = false;
  }
};

const confirmDelete = (admin: Admin) => {
  adminToDelete.value = admin;
  deleteDialog.value = true;
};

const deleteAdmin = async () => {
  if (!adminToDelete.value) return;

  try {
    deleting.value = true;
    await api.delete(`/users/${adminToDelete.value.id}`);
    showSnackbar("User deleted successfully", "success");
    deleteDialog.value = false;
    adminToDelete.value = null;
    fetchAdmins();
  } catch (_error) {
    showSnackbar("Failed to delete user", "error");
  } finally {
    deleting.value = false;
  }
};

const resetForm = () => {
  newAdmin.username = "";
  newAdmin.email = "";
  newAdmin.password = "";
};

const showSnackbar = (text: string, color: string) => {
  snackbarText.value = text;
  snackbarColor.value = color;
  snackbar.value = true;
};

onMounted(() => {
  fetchAdmins();
});
</script>


