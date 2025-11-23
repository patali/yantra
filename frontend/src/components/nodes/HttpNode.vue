<template>
  <div
    class="custom-node http-node"
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
        mdi-web
      </v-icon>
      <span class="node-label">{{ data.label || 'HTTP Request' }}</span>
    </div>

    <div class="node-body">
      <div class="node-method">
        {{ data.config?.method || 'GET' }}
      </div>
      <div class="node-url">
        {{ truncateUrl(data.config?.url) }}
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

const truncateUrl = (url?: string) => {
  if (!url) return "No URL set";
  if (url.length > 30) return url.substring(0, 27) + "...";
  return url;
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

.node-method {
  font-weight: bold;
  color: #2196F3;
  margin-bottom: 4px;
}

.node-url {
  word-break: break-all;
  color: rgb(var(--v-theme-on-surface));
}

.handle {
  width: 10px;
  height: 10px;
  background: #2196F3;
  border: 2px solid rgb(var(--v-theme-surface));
}
</style>

