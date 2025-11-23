import { ref, onUnmounted, type Ref } from "vue";

export interface SSEOptions {
  /**
   * URL for the SSE endpoint (relative or absolute)
   */
  url: string;

  /**
   * Maximum number of reconnection attempts (default: 5)
   */
  maxReconnectAttempts?: number;

  /**
   * Base delay for reconnection in ms (default: 5000)
   * Uses exponential backoff: delay * attemptNumber
   */
  reconnectDelay?: number;

  /**
   * Maximum reconnect delay in ms (default: 30000)
   */
  maxReconnectDelay?: number;

  /**
   * Event handlers for SSE events
   */
  onMessage?: Record<string, (data: any) => void>;

  /**
   * Callback when connection is established
   */
  onConnected?: () => void;

  /**
   * Callback when connection is closed
   */
  onClosed?: () => void;

  /**
   * Callback when an error occurs
   */
  onError?: (error: string) => void;

  /**
   * Callback when max reconnection attempts are reached
   */
  onMaxReconnectAttemptsReached?: () => void;
}

export interface SSEConnection {
  /**
   * Whether the connection is currently active
   */
  isConnected: Ref<boolean>;

  /**
   * Current reconnection attempt number
   */
  reconnectAttempts: Ref<number>;

  /**
   * Start the SSE connection
   */
  connect: () => void;

  /**
   * Close the SSE connection
   */
  disconnect: () => void;

  /**
   * Force reconnect (resets attempt counter)
   */
  reconnect: () => void;
}

/**
 * Composable for Server-Sent Events (SSE) connections
 *
 * Features:
 * - Automatic reconnection with exponential backoff
 * - Event-based message handling
 * - Automatic cleanup on component unmount
 * - Type-safe event handling
 *
 * @example
 * ```typescript
 * const { isConnected, reconnectAttempts, connect, disconnect } = useSSE({
 *   url: `/api/workflows/${workflowId}/executions/${executionId}/stream`,
 *   onMessage: {
 *     update: (data) => console.log('Update:', data),
 *     complete: (data) => console.log('Complete:', data),
 *   },
 *   onConnected: () => console.log('Connected!'),
 *   onError: (error) => console.error('Error:', error),
 * });
 *
 * // Start connection
 * connect();
 *
 * // Later: disconnect
 * disconnect();
 * ```
 */
export function useSSE(options: SSEOptions): SSEConnection {
  const {
    url,
    maxReconnectAttempts = 5,
    reconnectDelay = 5000,
    maxReconnectDelay = 30000,
    onMessage = {},
    onConnected,
    onClosed,
    onError,
    onMaxReconnectAttemptsReached,
  } = options;

  let eventSource: EventSource | null = null;
  const isConnected = ref(false);
  const reconnectAttempts = ref(0);
  let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;

  /**
   * Build the SSE URL with authentication token
   */
  const buildUrl = (): string => {
    const token = localStorage.getItem("token");
    if (!token) {
      throw new Error("No authentication token found");
    }

    const separator = url.includes("?") ? "&" : "?";
    return `${url}${separator}token=${encodeURIComponent(token)}`;
  };

  /**
   * Clean up existing connection and timers
   */
  const cleanup = () => {
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout);
      reconnectTimeout = null;
    }

    if (eventSource) {
      eventSource.close();
      eventSource = null;
    }

    isConnected.value = false;
  };

  /**
   * Schedule a reconnection attempt
   */
  const scheduleReconnect = () => {
    if (reconnectAttempts.value >= maxReconnectAttempts) {
      console.error("Max SSE reconnection attempts reached");
      onMaxReconnectAttemptsReached?.();
      onError?.("Max reconnection attempts reached. Please refresh the page.");
      return;
    }

    reconnectAttempts.value++;
    const delay = Math.min(
      reconnectDelay * reconnectAttempts.value,
      maxReconnectDelay
    );

    reconnectTimeout = setTimeout(() => {
      connect();
    }, delay);
  };

  /**
   * Connect to the SSE endpoint
   */
  const connect = () => {
    try {
      // Clean up existing connection
      cleanup();

      // Build URL with token
      const sseUrl = buildUrl();

      // Create EventSource connection
      eventSource = new EventSource(sseUrl);

      // Handle connection open
      eventSource.addEventListener("open", () => {
        isConnected.value = true;
      });

      // Handle 'connected' event from server
      eventSource.addEventListener("connected", () => {
        reconnectAttempts.value = 0; // Reset on successful connection
        onConnected?.();
      });

      // Handle custom message events
      Object.entries(onMessage).forEach(([eventType, handler]) => {
        eventSource!.addEventListener(eventType, (event: MessageEvent) => {
          try {
            const data = JSON.parse(event.data);
            handler(data);
          } catch (error) {
            console.error(`Error parsing SSE event '${eventType}':`, error);
          }
        });
      });

      // Handle errors and disconnections
      eventSource.onerror = () => {
        const readyState = eventSource?.readyState;
        console.error("SSE connection error, readyState:", readyState);

        if (readyState === EventSource.CLOSED) {
          isConnected.value = false;
          onClosed?.();

          // Attempt to reconnect
          scheduleReconnect();
        } else if (readyState === EventSource.CONNECTING) {
          // Still connecting, wait
        }
      };
    } catch (error) {
      console.error("Error creating SSE connection:", error);
      const errorMessage = error instanceof Error ? error.message : "Unknown error";
      onError?.(errorMessage);
    }
  };

  /**
   * Disconnect from the SSE endpoint
   */
  const disconnect = () => {
    cleanup();
    onClosed?.();
  };

  /**
   * Force reconnect (resets attempt counter)
   */
  const reconnect = () => {
    reconnectAttempts.value = 0;
    disconnect();
    connect();
  };

  // Cleanup on component unmount
  onUnmounted(() => {
    disconnect();
  });

  return {
    isConnected,
    reconnectAttempts,
    connect,
    disconnect,
    reconnect,
  };
}
