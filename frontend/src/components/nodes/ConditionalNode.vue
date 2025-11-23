<template>
  <div
    class="custom-node conditional-node"
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
        mdi-call-split
      </v-icon>
      <span class="node-label">{{ data.label || 'Conditional' }}</span>
    </div>

    <div class="node-body">
      <div class="node-condition">
        {{ conditionText }}
      </div>
    </div>

    <div class="output-handles">
      <Handle
        id="true"
        type="source"
        :position="Position.Bottom"
        :style="{ left: '30%' }"
        class="handle handle-true"
      />
      <Handle
        id="false"
        type="source"
        :position="Position.Bottom"
        :style="{ left: '70%' }"
        class="handle handle-false"
      />
    </div>

    <div class="handle-labels">
      <span class="label-true">True</span>
      <span class="label-false">False</span>
    </div>
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

const conditionText = computed(() => {
  const conditions = props.data.config?.conditions;
  if (!conditions || conditions.length === 0) return "No conditions";
  const count = conditions.length;
  const operator = props.data.config?.logicalOperator || "AND";
  return `${count} condition${count > 1 ? "s" : ""} (${operator})`;
});
</script>

<style scoped>
.custom-node {
  background: rgb(var(--v-theme-surface));
  border: 2px solid #4caf50;
  border-radius: 8px;
  padding: 10px;
  min-width: 200px;
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

.conditional-node {
  border-color: #4caf50;
  min-width: 200px;
}

.conditional-node .node-header {
  color: #4caf50;
}

.node-condition {
  text-align: center;
  font-size: 11px;
  color: rgb(var(--v-theme-on-surface));
}

.output-handles {
  position: relative;
  height: 0;
}

.handle-labels {
  display: flex;
  justify-content: space-around;
  margin-top: 8px;
  font-size: 10px;
  color: rgb(var(--v-theme-on-surface));
}

.label-true {
  margin-left: -10px;
}

.label-false {
  margin-right: -10px;
}

.handle {
  width: 10px;
  height: 10px;
  border: 2px solid rgb(var(--v-theme-surface));
}

.handle-true {
  background: #4caf50;
}

.handle-false {
  background: #f44336;
}
</style>

