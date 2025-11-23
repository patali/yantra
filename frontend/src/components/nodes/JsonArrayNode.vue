<template>
  <div
    class="custom-node json-array-node"
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
        color="orange"
      >
        mdi-code-json
      </v-icon>
      <span class="node-label">{{ data.label || 'JSON Array' }}</span>
    </div>

    <div class="node-body">
      <div class="node-count">
        {{ getArrayCount() }} items
      </div>
      <div class="node-info">
        {{ getArrayPreview() }}
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

const props = defineProps<Props>();

const getArrayCount = () => {
  try {
    const jsonArray = props.data.config?.jsonArray;
    if (!jsonArray) return 0;
    const parsed = JSON.parse(jsonArray);
    return Array.isArray(parsed) ? parsed.length : 0;
  } catch {
    return 0;
  }
};

const getArrayPreview = () => {
  try {
    const jsonArray = props.data.config?.jsonArray;
    if (!jsonArray) return "No data";
    const parsed = JSON.parse(jsonArray);
    if (!Array.isArray(parsed) || parsed.length === 0) return "Empty array";

    const firstItem = parsed[0];
    if (typeof firstItem === "object" && firstItem !== null) {
      const keys = Object.keys(firstItem);
      if (keys.length > 3) {
        return keys.slice(0, 3).join(", ") + "...";
      }
      return keys.join(", ");
    }
    return "Invalid format";
  } catch {
    return "Invalid JSON";
  }
};
</script>

<style scoped>
.custom-node {
  background: rgb(var(--v-theme-surface));
  border: 2px solid #FF9800;
  border-radius: 8px;
  padding: 10px;
  min-width: 180px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  color: rgb(var(--v-theme-on-surface));
}

.custom-node.selected {
  border-color: #F57C00;
  box-shadow: 0 4px 8px rgba(0,0,0,0.2);
}

.node-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  font-weight: 600;
  color: #FF9800;
}

.node-label {
  font-size: 14px;
}

.node-body {
  font-size: 12px;
  color: rgb(var(--v-theme-on-surface-variant));
}

.node-count {
  font-weight: bold;
  color: #FF9800;
  margin-bottom: 4px;
}

.node-info {
  word-break: break-all;
  color: rgb(var(--v-theme-on-surface));
  font-size: 11px;
}

.handle {
  width: 10px;
  height: 10px;
  background: #FF9800;
  border: 2px solid rgb(var(--v-theme-surface));
}
</style>
