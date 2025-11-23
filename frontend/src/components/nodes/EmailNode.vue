<template>
  <div
    class="custom-node email-node"
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
        mdi-email
      </v-icon>
      <span class="node-label">{{ data.label || 'Send Email' }}</span>
    </div>

    <div class="node-body">
      <div class="node-to">
        To: {{ truncate(data.config?.to) }}
      </div>
      <div class="node-subject">
        {{ truncate(data.config?.subject) }}
      </div>
      <div
        v-if="data.config?.provider"
        class="node-provider"
      >
        Provider: {{ data.config.provider }}
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

const truncate = (text?: string | string[]) => {
  if (!text) return "Not set";
  const str = Array.isArray(text) ? text[0] : text;
  if (str.length > 25) return str.substring(0, 22) + "...";
  return str;
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

.email-node {
  border-color: #2196F3;
}

.email-node .node-header {
  color: #2196F3;
}

.node-to {
  font-weight: bold;
  margin-bottom: 4px;
  color: rgb(var(--v-theme-on-surface));
}

.node-subject {
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

