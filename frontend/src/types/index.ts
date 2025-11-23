export interface User {
  id: string;
  username: string;
  email: string;
  theme?: string;
  createdBy?: string;
  createdAt: string;
  updatedAt?: string;
}

// Legacy alias for compatibility
export type Admin = User;

// Workflow Types
export interface WorkflowNode {
  id: string;
  type: string;
  position: { x: number; y: number };
  data: {
    label: string;
    config: Record<string, any>;
  };
}

export interface WorkflowEdge {
  id: string;
  source: string;
  target: string;
  sourceHandle?: string;
  targetHandle?: string;
}

export interface WorkflowDefinition {
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
}

export interface Workflow {
  id: string;
  name: string;
  description?: string;
  isActive: boolean;
  schedule?: string;
  timezone: string;
  currentVersion: number;
  createdBy: string;
  creator?: {
    username: string;
    email: string;
  };
  createdAt: string;
  updatedAt: string;
  _count?: {
    executions: number;
    versions: number;
  };
  versions?: WorkflowVersion[];
}

export interface WorkflowVersion {
  id: string;
  workflowId: string;
  version: number;
  definition: string;
  changeLog?: string;
  createdAt: string;
}

export interface WorkflowExecution {
  id: string;
  workflowId: string;
  workflow?: Workflow;
  version: number;
  status: "queued" | "running" | "success" | "error" | "cancelled" | "partially_failed" | "interrupted";
  triggerType: "manual" | "scheduled" | "webhook";
  input?: string;
  output?: string;
  error?: string;
  startedAt: string;
  completedAt?: string;
  _count?: {
    nodeExecutions: number;
  };
  nodeExecutions?: WorkflowNodeExecution[];
}

export interface WorkflowNodeExecution {
  id: string;
  executionId: string;
  nodeId: string;
  nodeType: string;
  status: "pending" | "running" | "success" | "error" | "skipped";
  input?: string;
  output?: string;
  error?: string;
  startedAt: string;
  completedAt?: string;
  canRetry?: boolean;
  retryCount?: number;
  parentLoopNodeId?: string; // Node ID of parent loop (if this execution is part of a loop body)
}

// Dead letter message interface
export interface DeadLetterMessage {
  id: string;
  nodeExecutionId: string;
  eventType: string;
  payload: string;
  status: string;
  attempts: number;
  maxAttempts: number;
  lastError?: string;
  lastAttemptAt?: string;
  createdAt: string;
  nodeExecution?: WorkflowNodeExecution;
}

// Recovery operations interface
export interface RecoveryOptions {
  canRestartWorkflow: boolean;
  canRetryNodes: string[]; // Array of node IDs that can be retried
  deadLetterMessages: DeadLetterMessage[];
}

