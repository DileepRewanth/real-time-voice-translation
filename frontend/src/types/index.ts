export interface TranslationRequest {
  text: string;
  engine: 'gemini' | 'mymemory';
  context?: string[];
  tone_preference?: 'formal' | 'casual' | '';
}

export interface LatencyBreakdown {
  pre_process_ms: number;
  translate_ms: number;
  post_process_ms: number;
  total_ms: number;
}

export interface TranslationResult {
  original_text: string;
  processed_text: string;
  translated_text: string;
  engine: 'gemini' | 'mymemory';
  cached: boolean;
  latency: LatencyBreakdown;
  timestamp: string;
}

export interface WSMessage {
  type: 'translate' | 'status' | 'translation' | 'error' | 'pong' | 'ping';
  payload: unknown;
}

export interface WSStatusPayload {
  stage: PipelineStage;
  status: 'processing' | 'completed' | 'error';
  message?: string;
}

export type PipelineStage = 'pre_process' | 'translate' | 'post_process';

export type AppState = 'idle' | 'listening' | 'processing' | 'speaking' | 'error';

export interface HistoryEntry {
  id: string;
  originalText: string;
  translatedText: string;
  engine: 'gemini' | 'mymemory';
  latency: LatencyBreakdown;
  timestamp: Date;
  cached: boolean;
}

export interface Settings {
  engine: 'gemini' | 'mymemory';
  geminiApiKey: string;
  tonePreference: 'formal' | 'casual' | '';
  ttsRate: number;
  ttsPitch: number;
  autoSpeak: boolean;
}

export interface HealthStatus {
  status: string;
  timestamp: string;
  services: Record<string, string>;
  version: string;
}

export interface ConfigResponse {
  available_engines: string[];
  default_engine: string;
  gemini_configured: boolean;
  websocket_path: string;
}
