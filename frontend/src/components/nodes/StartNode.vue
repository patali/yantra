<template>
  <div
    class="custom-node start-node"
    :class="{ selected: selected }"
  >
    <div class="node-header">
      <v-icon
        size="small"
        color="green"
      >
        mdi-play-circle
      </v-icon>
      <span class="node-label">Start</span>
    </div>
    <div
      v-if="data.config?.triggerType && data.config.triggerType !== 'manual'"
      class="node-trigger-info"
    >
      <v-chip
        size="x-small"
        variant="flat"
        :color="getTriggerColor(data.config.triggerType)"
      >
        {{ getTriggerLabel(data.config.triggerType) }}
      </v-chip>
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

const getTriggerLabel = (triggerType: string): string => {
  switch (triggerType) {
    case "webhook":
      return "Webhook";
    case "cron":
      return "Cron";
    case "both":
      return "Webhook + Cron";
    default:
      return "";
  }
};

const getTriggerColor = (triggerType: string): string => {
  switch (triggerType) {
    case "webhook":
      return "blue";
    case "cron":
      return "orange";
    case "both":
      return "purple";
    default:
      return "green";
  }
};
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

.start-node {
  border-color: #4caf50;
  min-width: 100px;
}

.start-node .node-header {
  color: #4caf50;
  justify-content: center;
}

.node-trigger-info {
  display: flex;
  justify-content: center;
  margin-top: 4px;
}

.handle {
  width: 10px;
  height: 10px;
  background: #4caf50;
  border: 2px solid rgb(var(--v-theme-surface));
}
</style>

