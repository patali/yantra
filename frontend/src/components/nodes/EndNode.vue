<template>
  <div
    class="custom-node end-node"
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
        color="green"
      >
        mdi-stop-circle
      </v-icon>
      <span class="node-label">End</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Handle, Position } from "@vue-flow/core";

interface Props {
  id: string;
  data: {
    label: string;
    config: Record<string, any>;
  };
  selected?: boolean;
}

defineProps<Props>();
</script>

<style scoped>
.custom-node {
  background: rgb(var(--v-theme-surface));
  border: 2px solid #4caf50;
  border-radius: 8px;
  padding: 10px;
  min-width: 100px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  color: rgb(var(--v-theme-on-surface));
}

.custom-node.selected {
  border-color: #388e3c;
  box-shadow: 0 4px 8px rgba(0,0,0,0.2);
}

.node-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: #4caf50;
  justify-content: center;
}

.node-label {
  font-size: 14px;
}

.end-node {
  border-color: #4caf50;
  min-width: 100px;
}

.end-node .node-header {
  color: #4caf50;
  justify-content: center;
}

.handle {
  width: 10px;
  height: 10px;
  background: #4caf50;
  border: 2px solid rgb(var(--v-theme-surface));
}
</style>

