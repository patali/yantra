<template>
  <div class="sleep-properties">
    <v-select
      v-model="localConfig.mode"
      label="Sleep Mode"
      :items="modeOptions"
      item-title="text"
      item-value="value"
      variant="outlined"
      density="compact"
      class="mb-4"
      @update:model-value="emitUpdate"
    />

    <!-- Relative Mode Configuration -->
    <template v-if="localConfig.mode === 'relative'">
      <v-row class="mb-2">
        <v-col cols="7">
          <v-text-field
            v-model.number="localConfig.duration_value"
            label="Duration"
            hint="How long to sleep"
            persistent-hint
            variant="outlined"
            density="compact"
            type="number"
            min="0"
            @update:model-value="emitUpdate"
          />
        </v-col>
        <v-col cols="5">
          <v-select
            v-model="localConfig.duration_unit"
            label="Unit"
            :items="durationUnits"
            variant="outlined"
            density="compact"
            @update:model-value="emitUpdate"
          />
        </v-col>
      </v-row>

      <v-row class="mb-4">
        <v-col cols="12">
          <v-chip-group>
            <v-chip
              variant="outlined"
              size="small"
              @click="setRelativeDuration(30, 'seconds')"
            >
              30 seconds
            </v-chip>
            <v-chip
              variant="outlined"
              size="small"
              @click="setRelativeDuration(5, 'minutes')"
            >
              5 minutes
            </v-chip>
            <v-chip
              variant="outlined"
              size="small"
              @click="setRelativeDuration(1, 'hours')"
            >
              1 hour
            </v-chip>
            <v-chip
              variant="outlined"
              size="small"
              @click="setRelativeDuration(1, 'days')"
            >
              1 day
            </v-chip>
          </v-chip-group>
        </v-col>
      </v-row>

      <v-autocomplete
        v-model="localConfig.timezone"
        label="Timezone (optional)"
        hint="Timezone for duration calculation"
        persistent-hint
        :items="timezones"
        item-title="title"
        item-value="value"
        variant="outlined"
        density="compact"
        class="mb-4"
        clearable
        auto-select-first
        @update:model-value="emitUpdate"
      />
    </template>

    <!-- Absolute Mode Configuration -->
    <template v-if="localConfig.mode === 'absolute'">
      <v-text-field
        v-model="localConfig.target_date"
        label="Target Date/Time"
        hint="ISO 8601 format: 2024-12-31T23:59:59 or 2024-12-31"
        persistent-hint
        variant="outlined"
        density="compact"
        type="datetime-local"
        class="mb-4"
        @update:model-value="emitUpdate"
      />

      <v-autocomplete
        v-model="localConfig.timezone"
        label="Timezone"
        hint="Timezone for target date"
        persistent-hint
        :items="timezones"
        item-title="title"
        item-value="value"
        variant="outlined"
        density="compact"
        class="mb-4"
        clearable
        auto-select-first
        @update:model-value="emitUpdate"
      />

      <v-row class="mb-4">
        <v-col cols="12">
          <v-chip-group>
            <v-chip
              variant="outlined"
              size="small"
              @click="setAbsoluteDate('tomorrow')"
            >
              Tomorrow
            </v-chip>
            <v-chip
              variant="outlined"
              size="small"
              @click="setAbsoluteDate('next-week')"
            >
              Next Week
            </v-chip>
            <v-chip
              variant="outlined"
              size="small"
              @click="setAbsoluteDate('next-month')"
            >
              Next Month
            </v-chip>
          </v-chip-group>
        </v-col>
      </v-row>
    </template>

    <v-alert
      type="info"
      variant="tonal"
      density="compact"
      class="mt-4"
    >
      <div class="text-body-2">
        <strong>Sleep Node:</strong> Pauses workflow execution until a specific time or for a duration.
      </div>
      <div class="text-caption mt-2">
        <strong>Relative mode:</strong> Sleeps for a specified duration from the current time.
        <br>
        <strong>Absolute mode:</strong> Sleeps until a specific date/time.
      </div>
      <div class="text-caption mt-2">
        If the target time has already passed, the workflow continues immediately.
      </div>
    </v-alert>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { getTimezonesWithBrowser, getBrowserTimezone } from "@/constants/timezones";

interface Props {
  modelValue: Record<string, any>;
}

const props = defineProps<Props>();
const emit = defineEmits<{
  (e: "update:modelValue", value: Record<string, any>): void;
  (e: "update"): void;
}>();

const modeOptions = [
  { text: "Relative (sleep for a duration)", value: "relative" },
  { text: "Absolute (sleep until a specific time)", value: "absolute" },
];

const durationUnits = [
  { title: "Seconds", value: "seconds" },
  { title: "Minutes", value: "minutes" },
  { title: "Hours", value: "hours" },
  { title: "Days", value: "days" },
  { title: "Weeks", value: "weeks" },
];

// Get browser's local timezone as default
const defaultTimezone = getBrowserTimezone();

// Get timezones list with browser timezone included
const timezones = getTimezonesWithBrowser();

const localConfig = ref({
  mode: props.modelValue?.mode || "relative",
  // Relative mode fields
  duration_value: props.modelValue?.duration_value || 1,
  duration_unit: props.modelValue?.duration_unit || "hours",
  // Absolute mode fields
  target_date: props.modelValue?.target_date || "",
  // Common fields - use browser timezone as default
  timezone: props.modelValue?.timezone || defaultTimezone,
});

watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue) {
      localConfig.value = {
        mode: newValue.mode || "relative",
        duration_value: newValue.duration_value || 1,
        duration_unit: newValue.duration_unit || "hours",
        target_date: newValue.target_date || "",
        timezone: newValue.timezone || defaultTimezone,
      };
    }
  },
  { deep: true }
);

const emitUpdate = () => {
  emit("update:modelValue", localConfig.value);
  emit("update");
};

const setRelativeDuration = (value: number, unit: string) => {
  localConfig.value.duration_value = value;
  localConfig.value.duration_unit = unit;
  emitUpdate();
};

const setAbsoluteDate = (preset: string) => {
  const now = new Date();
  let targetDate = new Date();

  switch (preset) {
    case "tomorrow":
      targetDate.setDate(now.getDate() + 1);
      targetDate.setHours(9, 0, 0, 0);
      break;
    case "next-week":
      targetDate.setDate(now.getDate() + 7);
      targetDate.setHours(9, 0, 0, 0);
      break;
    case "next-month":
      targetDate.setMonth(now.getMonth() + 1);
      targetDate.setHours(9, 0, 0, 0);
      break;
  }

  // Format as ISO 8601 string without timezone (for datetime-local input)
  const year = targetDate.getFullYear();
  const month = String(targetDate.getMonth() + 1).padStart(2, "0");
  const day = String(targetDate.getDate()).padStart(2, "0");
  const hours = String(targetDate.getHours()).padStart(2, "0");
  const minutes = String(targetDate.getMinutes()).padStart(2, "0");

  localConfig.value.target_date = `${year}-${month}-${day}T${hours}:${minutes}`;
  emitUpdate();
};
</script>

<style scoped>
.sleep-properties {
  padding: 0;
}
</style>
