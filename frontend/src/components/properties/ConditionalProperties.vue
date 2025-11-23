<template>
  <v-form>
    <div class="mb-4">
      <div class="d-flex justify-space-between align-center mb-2">
        <span class="text-subtitle-2">Conditions</span>
        <v-btn
          size="x-small"
          icon="mdi-plus"
          variant="text"
          @click="addCondition"
        />
      </div>

      <div
        v-for="(condition, index) in config.conditions"
        :key="index"
        class="mb-3 pa-2 condition-card"
      >
        <div class="d-flex justify-space-between mb-2">
          <span class="text-caption">Condition {{ index + 1 }}</span>
          <v-btn
            size="x-small"
            icon="mdi-delete"
            variant="text"
            @click="removeCondition(index)"
          />
        </div>

        <v-text-field
          v-model="condition.left"
          label="Left Value"
          placeholder="input.field"
          variant="outlined"
          density="compact"
          class="mb-2"
          @update:model-value="emitUpdate"
        />

        <v-select
          v-model="condition.operator"
          label="Operator"
          :items="operators"
          variant="outlined"
          density="compact"
          class="mb-2"
          @update:model-value="emitUpdate"
        />

        <v-text-field
          v-model="condition.right"
          label="Right Value"
          placeholder="value"
          variant="outlined"
          density="compact"
          @update:model-value="emitUpdate"
        />
      </div>
    </div>

    <v-select
      v-model="config.logicalOperator"
      label="Logical Operator"
      :items="['AND', 'OR']"
      variant="outlined"
      density="compact"
      @update:model-value="emitUpdate"
    />
  </v-form>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";

interface Props {
  modelValue: Record<string, any>;
}

const props = defineProps<Props>();
const emit = defineEmits(["update:modelValue"]);

const operators = [
  { title: "Equals", value: "eq" },
  { title: "Not Equals", value: "neq" },
  { title: "Greater Than", value: "gt" },
  { title: "Less Than", value: "lt" },
  { title: "Greater or Equal", value: "gte" },
  { title: "Less or Equal", value: "lte" },
  { title: "Contains", value: "contains" },
  { title: "Exists", value: "exists" },
];

const config = ref(props.modelValue || {
  conditions: [],
  logicalOperator: "AND",
});

const addCondition = () => {
  config.value.conditions.push({
    left: "",
    operator: "eq",
    right: "",
  });
  emitUpdate();
};

const removeCondition = (index: number) => {
  config.value.conditions.splice(index, 1);
  emitUpdate();
};

const emitUpdate = () => {
  emit("update:modelValue", config.value);
};

watch(() => props.modelValue, (newVal) => {
  config.value = newVal;
}, { deep: true });
</script>

<style scoped>
.condition-card {
  border: 1px solid rgb(var(--v-theme-surface-variant));
  border-radius: 4px;
  background: rgb(var(--v-theme-surface));
}
</style>

