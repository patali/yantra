import axios from "axios";

const api = axios.create({
  baseURL: "/api",
  headers: {
    "Content-Type": "application/json",
  },
});

// Add token to requests if it exists
const token = localStorage.getItem("token");
if (token) {
  api.defaults.headers.common["Authorization"] = `Bearer ${token}`;
}

// Add request interceptor to ensure token is always up to date
api.interceptors.request.use((config) => {
  const currentToken = localStorage.getItem("token");
  if (currentToken) {
    config.headers.Authorization = `Bearer ${currentToken}`;
  }
  return config;
});

// Add response interceptor to handle auth errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Token expired or invalid
      localStorage.removeItem("token");
      delete api.defaults.headers.common["Authorization"];
      // Redirect to login if not already there
      if (window.location.pathname !== "/login") {
        window.location.href = "/login";
      }
    }
    return Promise.reject(error);
  }
);

// Recovery operations
export const recoveryApi = {
  // All workflow runs
  getAllRuns: (status: string = "all") => api.get(`/recovery/runs?status=${status}`),

  // Dead letter queue operations (for async node failures)
  getDeadLetterMessages: () => api.get("/recovery/dead-letter"),
  retryDeadLetterMessage: (messageId: string) => api.post(`/recovery/dead-letter/${messageId}/retry`),

  // Workflow restart operations
  restartWorkflow: (executionId: string) => api.post(`/recovery/workflows/${executionId}/restart`),

  // Workflow resumption from checkpoint
  resumeWorkflow: (workflowId: string, executionId: string) => api.post(`/workflows/${workflowId}/executions/${executionId}/resume`),

  // Workflow cancellation
  cancelWorkflow: (executionId: string) => api.post(`/recovery/workflows/${executionId}/cancel`),

  // Node re-execution operations
  reExecuteNode: (executionId: string, nodeId: string) => api.post(`/recovery/executions/${executionId}/nodes/${nodeId}/retry`),

  // Get execution details with recovery options
  getExecutionWithRecovery: (workflowId: string, executionId: string) => api.get(`/workflows/${workflowId}/executions/${executionId}?includeRecovery=true`),
};

export default api;

