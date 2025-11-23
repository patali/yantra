<template>
  <div
    class="custom-node slack-node"
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
        color="blue"
      >
        mdi-slack
      </v-icon>
      <span class="node-label">{{ data.label || 'Slack Message' }}</span>
    </div>

    <div class="node-body">
      <div class="node-channel">
        {{ truncate(data.config?.channel) }}
      </div>
      <div class="node-message">
        {{ truncate(data.config?.message) }}
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

const truncate = (text?: string) => {
  if (!text) return "Not set";
  if (text.length > 25) return text.substring(0, 22) + "...";
  return text;
};
</script>

<style scoped>
.custom-node {
  background: rgb(var(--v-theme-surface));
  border: 2px solid #2196F3;
  border-radius: 8px;
  padding: 10px;
  min-width: 180px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  color: rgb(var(--v-theme-on-surface));
}

.custom-node.selected {
  border-color: #1976D2;
  box-shadow: 0 4px 8px rgba(0,0,0,0.2);
}

.node-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  font-weight: 600;
  color: #2196F3;
}

.node-label {
  font-size: 14px;
}

.node-body {
  font-size: 12px;
  color: rgb(var(--v-theme-on-surface-variant));
}

.slack-node {
  border-color: #2196F3;
}

.slack-node .node-header {
  color: #2196F3;
}

.node-channel {
  font-weight: bold;
  margin-bottom: 4px;
  color: rgb(var(--v-theme-on-surface));
}

.node-message {
  font-style: italic;
  color: rgb(var(--v-theme-on-surface));
}

.handle {
  width: 10px;
  height: 10px;
  background: #2196F3;
  border: 2px solid rgb(var(--v-theme-surface));
}
</style>

