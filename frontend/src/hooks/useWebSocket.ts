import { useState, useCallback, useRef, useEffect } from 'react';
import type { WSMessage, WSStatusPayload, TranslationResult } from '../types';
import { getWebSocketURL } from '../services/api';

interface UseWebSocketReturn {
  isConnected: boolean;
  sendTranslation: (text: string, engine: string, context?: string[], tone?: string) => void;
  lastResult: TranslationResult | null;
  lastStatus: WSStatusPayload | null;
  lastError: string | null;
  reconnect: () => void;
}

export function useWebSocket(): UseWebSocketReturn {
  const [isConnected, setIsConnected] = useState(false);
  const [lastResult, setLastResult] = useState<TranslationResult | null>(null);
  const [lastStatus, setLastStatus] = useState<WSStatusPayload | null>(null);
  const [lastError, setLastError] = useState<string | null>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<number | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const maxReconnectAttempts = 10;

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) return;

    try {
      const url = getWebSocketURL();
      const ws = new WebSocket(url);

      ws.onopen = () => {
        setIsConnected(true);
        setLastError(null);
        reconnectAttemptsRef.current = 0;
      };

      ws.onmessage = (event: MessageEvent) => {
        try {
          const msg: WSMessage = JSON.parse(event.data);

          switch (msg.type) {
            case 'translation':
              setLastResult(msg.payload as TranslationResult);
              setLastStatus(null);
              break;
            case 'status':
              setLastStatus(msg.payload as WSStatusPayload);
              break;
            case 'error': {
              const errorPayload = msg.payload as { message: string };
              setLastError(errorPayload.message);
              break;
            }
            case 'pong':
              // Heartbeat response — no action needed
              break;
          }
        } catch {
          console.error('Failed to parse WebSocket message');
        }
      };

      ws.onclose = () => {
        setIsConnected(false);
        wsRef.current = null;

        // Auto-reconnect with exponential backoff
        if (reconnectAttemptsRef.current < maxReconnectAttempts) {
          const delay = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000);
          reconnectAttemptsRef.current++;
          reconnectTimeoutRef.current = window.setTimeout(connect, delay);
        }
      };

      ws.onerror = () => {
        setLastError('WebSocket connection error');
      };

      wsRef.current = ws;
    } catch {
      setLastError('Failed to create WebSocket connection');
    }
  }, []);

  const reconnect = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close();
    }
    reconnectAttemptsRef.current = 0;
    connect();
  }, [connect]);

  useEffect(() => {
    connect();

    // Heartbeat ping every 30 seconds
    const pingInterval = setInterval(() => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({
          type: 'ping',
          payload: { timestamp: Date.now() },
        }));
      }
    }, 30000);

    return () => {
      clearInterval(pingInterval);
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [connect]);

  const sendTranslation = useCallback((
    text: string,
    engine: string,
    context?: string[],
    tone?: string
  ) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      setLastError('WebSocket not connected');
      return;
    }

    const msg: WSMessage = {
      type: 'translate',
      payload: {
        text,
        engine,
        context: context || [],
        tone_preference: tone || '',
      },
    };

    wsRef.current.send(JSON.stringify(msg));
    setLastError(null);
  }, []);

  return {
    isConnected,
    sendTranslation,
    lastResult,
    lastStatus,
    lastError,
    reconnect,
  };
}
