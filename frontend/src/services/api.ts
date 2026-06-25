import type { TranslationRequest, TranslationResult, HealthStatus, ConfigResponse } from '../types';

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

/**
 * REST API client — used as a fallback when WebSocket is unavailable.
 */
export async function translateText(req: TranslationRequest): Promise<TranslationResult> {
  const response = await fetch(`${API_BASE}/api/v1/translate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || `Translation failed with status ${response.status}`);
  }

  return response.json();
}

/**
 * Fetch backend health status.
 */
export async function getHealth(): Promise<HealthStatus> {
  const response = await fetch(`${API_BASE}/health`);
  if (!response.ok) throw new Error('Health check failed');
  return response.json();
}

/**
 * Fetch backend configuration.
 */
export async function getConfig(): Promise<ConfigResponse> {
  const response = await fetch(`${API_BASE}/api/v1/config`);
  if (!response.ok) throw new Error('Config fetch failed');
  return response.json();
}

/**
 * Get the WebSocket URL for the backend.
 */
export function getWebSocketURL(): string {
  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const apiURL = new URL(API_BASE);
  return `${wsProtocol}//${apiURL.host}/api/v1/ws`;
}
