<template>
  <div
    class="custom-node loop-node"
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
        mdi-repeat
      </v-icon>
      <span class="node-label">{{ data.label || 'Loop' }}</span>
    </div>

    <div class="node-body">
      <div class="node-array-path">
        {{ data.config?.arrayPath || 'input array' }}
      </div>
      <div class="node-variables">
        {{ data.config?.itemVariable || 'item' }}[{{ data.config?.indexVariable || 'index' }}]
      </div>
      <div
        v-if="data.config?.max_iterations"
        class="node-iterations"
      >
        Max: {{ data.config.max_iterations.toLocaleString() }}
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
  min-width: 180px;
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
  margin-bottom: 8px;
  font-weight: 600;
  color: #4caf50;
}

.node-label {
  font-size: 14px;
}

.node-body {
  font-size: 12px;
  color: rgb(var(--v-theme-on-surface-variant));
}

.loop-node {
  border-color: #4caf50;
}

.loop-node .node-header {
  color: #4caf50;
}

.node-array-path {
  font-weight: bold;
  margin-bottom: 4px;
  color: rgb(var(--v-theme-on-surface));
}

.node-variables {
  font-style: italic;
  font-size: 11px;
  color: rgb(var(--v-theme-on-surface));
}

.node-iterations {
  margin-top: 4px;
  font-size: 10px;
  font-weight: 600;
  color: #ff9800;
  background: rgba(255, 152, 0, 0.1);
  padding: 2px 6px;
  border-radius: 4px;
  display: inline-block;
}

.handle {
  width: 10px;
  height: 10px;
  background: #4caf50;
  border: 2px solid rgb(var(--v-theme-surface));
}
</style>

