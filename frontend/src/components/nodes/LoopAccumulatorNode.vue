<template>
  <div
    class="custom-node loop-accumulator-node"
    :class="{ selected: selected }"
  >
    <!-- Top Input: From previous node -->
    <Handle
      id="input"
      type="target"
      :position="Position.Top"
      class="handle"
    />

    <div class="node-header">
      <v-icon
        size="small"
        color="green"
      >
        mdi-format-list-bulleted-square
      </v-icon>
      <span class="node-label">{{ data.label || 'Loop Accumulator' }}</span>
    </div>

    <div class="node-body">
      <div class="node-array">
        {{ getArrayInfo() }}
      </div>
      <div class="node-variables">
        <span class="variable-badge">{{ data.config?.itemVariable || 'item' }}</span>
        <span class="variable-badge accumulator">{{ data.config?.accumulatorVariable || 'accumulated' }}</span>
      </div>
      <div
        v-if="data.config?.max_iterations"
        class="node-iterations"
      >
        Max: {{ data.config.max_iterations.toLocaleString() }}
      </div>
    </div>

    <!-- Left Side: Loop handles -->
    <!-- Loop Output: To loop body (top-left) -->
    <Handle
      id="loop-output"
      type="source"
      :position="Position.Left"
      class="handle handle-loop"
      :style="{ top: '35%' }"
    />
    <div class="handle-label handle-label-left handle-label-loop">
      To Loop
    </div>

    <!-- Accumulator Input: From loop body (bottom-left) -->
    <Handle
      id="accumulator-input"
      type="target"
      :position="Position.Left"
      class="handle handle-accumulator"
      :style="{ top: '65%' }"
    />
    <div class="handle-label handle-label-left handle-label-accumulator">
      From Loop
    </div>

    <!-- Right Output: Final result -->
    <Handle
      id="output"
      type="source"
      :position="Position.Right"
      class="handle"
      :style="{ top: '50%' }"
    />
    <div class="handle-label handle-label-right">
      Final Output
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

const props = defineProps<Props>();

const getArrayInfo = () => {
  const arrayPath = props.data.config?.arrayPath;
  if (!arrayPath) return "No array path set";
  return `Array: ${arrayPath}`;
};
</script>

<style scoped>
.custom-node {
  background: rgb(var(--v-theme-surface));
  border: 2px solid #4CAF50;
  border-radius: 8px;
  padding: 10px;
  min-width: 200px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  color: rgb(var(--v-theme-on-surface));
  position: relative;
}

.custom-node.selected {
  border-color: #388E3C;
  box-shadow: 0 4px 8px rgba(0,0,0,0.2);
}

.node-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  font-weight: 600;
  color: #4CAF50;
}

.node-label {
  font-size: 14px;
}

.node-body {
  font-size: 12px;
  color: rgb(var(--v-theme-on-surface-variant));
}

.node-array {
  margin-bottom: 8px;
  color: rgb(var(--v-theme-on-surface));
}

.node-variables {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.variable-badge {
  background: #4CAF50;
  color: white;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 500;
}

.variable-badge.accumulator {
  background: #FF9800;
}

.node-iterations {
  margin-top: 8px;
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
  background: #4CAF50;
  border: 2px solid rgb(var(--v-theme-surface));
}

.handle-loop {
  background: #2196F3;
}

.handle-accumulator {
  background: #FF9800;
}

.handle-label {
  position: absolute;
  font-size: 9px;
  font-weight: 600;
  white-space: nowrap;
  pointer-events: none;
  padding: 2px 4px;
  border-radius: 3px;
  background: rgba(0, 0, 0, 0.7);
  color: white;
  z-index: 10;
}

.handle-label-left {
  left: -60px;
}

.handle-label-loop {
  top: calc(35% - 10px);
  background: rgba(33, 150, 243, 0.9);
}

.handle-label-accumulator {
  top: calc(65% - 10px);
  background: rgba(255, 152, 0, 0.9);
}

.handle-label-right {
  right: -70px;
  top: calc(50% - 10px);
  background: rgba(76, 175, 80, 0.9);
}
</style>
