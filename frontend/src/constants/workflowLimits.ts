/**
 * Workflow Execution Limits
 *
 * These limits match the backend abuse prevention constants defined in:
 * yantra-server/src/services/workflow_engine.go
 *
 * Keep these values synchronized with the backend!
 */

export const WORKFLOW_LIMITS = {
  // Execution time limits
  MAX_EXECUTION_DURATION_MINUTES: 30,
  MAX_EXECUTION_DURATION_MS: 30 * 60 * 1000,

  // Node and structure limits
  MAX_TOTAL_NODES: 10000,
  MAX_LOOP_DEPTH: 5,

  // Iteration limits
  DEFAULT_MAX_ITERATIONS: 1000,
  GLOBAL_MAX_ITERATIONS: 10000,
  MIN_ITERATIONS: 1,

  // Data size limits (in bytes)
  MAX_ACCUMULATOR_SIZE: 10 * 1024 * 1024, // 10MB
  MAX_DATA_SIZE: 10 * 1024 * 1024, // 10MB

  // Delay limits (in milliseconds)
  MAX_ITERATION_DELAY: 5 * 60 * 1000, // 5 minutes
} as const;

/**
 * Human-readable limit descriptions for UI display
 */
export const LIMIT_DESCRIPTIONS = {
  MAX_EXECUTION_DURATION: "Maximum workflow execution time",
  MAX_TOTAL_NODES: "Maximum total nodes executed per workflow",
  MAX_LOOP_DEPTH: "Maximum nested loop depth",
  DEFAULT_MAX_ITERATIONS: "Default maximum iterations per loop",
  GLOBAL_MAX_ITERATIONS: "Global maximum iterations per loop",
  MAX_ACCUMULATOR_SIZE: "Maximum accumulated data size",
  MAX_DATA_SIZE: "Maximum input/output data size",
  MAX_ITERATION_DELAY: "Maximum delay between iterations",
} as const;

/**
 * Format bytes to human-readable string
 */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 Bytes";

  const k = 1024;
  const sizes = ["Bytes", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + " " + sizes[i];
}

/**
 * Format duration in milliseconds to human-readable string
 */
export function formatDuration(ms: number): string {
  const minutes = Math.floor(ms / 60000);
  const seconds = Math.floor((ms % 60000) / 1000);

  if (minutes > 0) {
    return seconds > 0 ? `${minutes}m ${seconds}s` : `${minutes}m`;
  }
  return `${seconds}s`;
}

/**
 * Validate iteration count against limits
 */
export function validateIterations(value: number | undefined): {
  valid: boolean;
  error?: string;
  warning?: string;
} {
  if (value === undefined || value === null) {
    return {
      valid: true,
      warning: `Using default limit: ${WORKFLOW_LIMITS.DEFAULT_MAX_ITERATIONS} iterations`,
    };
  }

  if (value < WORKFLOW_LIMITS.MIN_ITERATIONS) {
    return {
      valid: false,
      error: `Must be at least ${WORKFLOW_LIMITS.MIN_ITERATIONS}`,
    };
  }

  if (value > WORKFLOW_LIMITS.GLOBAL_MAX_ITERATIONS) {
    return {
      valid: false,
      error: `Cannot exceed global limit of ${WORKFLOW_LIMITS.GLOBAL_MAX_ITERATIONS}`,
    };
  }

  if (value > WORKFLOW_LIMITS.DEFAULT_MAX_ITERATIONS) {
    return {
      valid: true,
      warning: "High iteration count may impact performance",
    };
  }

  return { valid: true };
}
