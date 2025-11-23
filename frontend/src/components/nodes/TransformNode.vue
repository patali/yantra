<template>
  <div
    class="custom-node transform-node"
    :class="{ selected: selected }"
  >
    <Handle
      type="target"
      :position="Position.Top"
      class="handle"
    />

    <div class="node-header">
      <v-icon
        size="small"
        color="purple"
      >
        mdi-shuffle-variant
      </v-icon>
      <span class="node-label">{{ data.label || 'Transform' }}</span>
    </div>

    <div class="node-body">
      <div class="node-operations">
        {{ operationsText }}
      </div>
    </div>

    <Handle
      type="source"
      :position="Position.Bottom"
      class="handle"
    />
  </div>
</template>

<script setup lang="ts">
import { Handle, Position } from "@vue-flow/core";
import { computed } from "vue";

interface Props {
  id: string;
  data: {
    label: string;
    config: Record<string, any>;
  };
  selected?: boolean;
}

const props = defineProps<Props>();

const operationsText = computed(() => {
  const operations = props.data.config?.operations;
  if (!operations || operations.length === 0) return "No operations";
  const count = operations.length;
  return `${count} operation${count > 1 ? "s" : ""}`;
});
</script>

<style scoped>
.custom-node {
  background: rgb(var(--v-theme-surface));
  border: 2px solid #9c27b0;
  border-radius: 8px;
  padding: 10px;
  min-width: 180px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  color: rgb(var(--v-theme-on-surface));
}

.custom-node.selected {
  border-color: #7b1fa2;
  box-shadow: 0 4px 8px rgba(0,0,0,0.2);
}

.node-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  font-weight: 600;
  color: #9c27b0;
}

.node-label {
  font-size: 14px;
}

.node-body {
  font-size: 12px;
  color: rgb(var(--v-theme-on-surface-variant));
}

.transform-node {
  border-color: #9c27b0;
}

.transform-node .node-header {
  color: #9c27b0;
}

.node-operations {
  text-align: center;
  font-size: 11px;
  color: rgb(var(--v-theme-on-surface));
}

.handle {
  width: 10px;
  height: 10px;
  background: #9c27b0;
  border: 2px solid rgb(var(--v-theme-surface));
}
</style>

