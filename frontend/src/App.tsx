import { useState, useCallback, useEffect, useRef } from 'react';
import type { AppState, HistoryEntry, PipelineStage, LatencyBreakdown } from './types';
import { useASR } from './hooks/useASR';
import { useTTS } from './hooks/useTTS';
import { useWebSocket } from './hooks/useWebSocket';
import { useAudioVisualizer } from './hooks/useAudioVisualizer';
import { useSettings } from './hooks/useSettings';
import { SentenceAccumulator, removeFillers } from './services/textProcessor';
import Pipeline from './components/Pipeline';
import MicrophoneButton from './components/MicrophoneButton';
import TranscriptPanel from './components/TranscriptPanel';
import TranslationPanel from './components/TranslationPanel';
import AudioVisualizer from './components/AudioVisualizer';
import HistoryLog from './components/HistoryLog';
import SettingsPanel from './components/SettingsPanel';
import LatencyMonitor from './components/LatencyMonitor';
import StatusBadge from './components/StatusBadge';

const accumulator = new SentenceAccumulator(600);

function App() {
  const [appState, setAppState] = useState<AppState>('idle');
  const [processedText, setProcessedText] = useState('');
  const [translatedText, setTranslatedText] = useState('');
  const [currentEngine, setCurrentEngine] = useState('');
  const [isCached, setIsCached] = useState(false);
  const [latency, setLatency] = useState<LatencyBreakdown | null>(null);
  const [history, setHistory] = useState<HistoryEntry[]>([]);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [stageStatuses, setStageStatuses] = useState<Record<PipelineStage, 'idle' | 'processing' | 'completed' | 'error'>>({
    pre_process: 'idle',
    translate: 'idle',
    post_process: 'idle',
  });

  const [settings, updateSettings] = useSettings();
  const contextRef = useRef<string[]>([]);
  const flushTimerRef = useRef<number | null>(null);

  // WebSocket connection
  const { isConnected, sendTranslation, lastResult, lastStatus, lastError } = useWebSocket();

  // TTS
  const { speak, stop: stopTTS, isSpeaking } = useTTS();

  // Handle sending text for translation
  const handleTranslate = useCallback((text: string) => {
    if (!text.trim() || !isConnected) return;

    const cleaned = removeFillers(text);
    if (!cleaned || cleaned.length < 2) return;

    setProcessedText(cleaned);
    setAppState('processing');
    setStageStatuses({
      pre_process: 'processing',
      translate: 'idle',
      post_process: 'idle',
    });

    sendTranslation(
      cleaned,
      settings.engine,
      contextRef.current.slice(-5),
      settings.tonePreference
    );
  }, [isConnected, sendTranslation, settings.engine, settings.tonePreference]);

  // ASR with barge-in support
  const handleFinalResult = useCallback((text: string) => {
    // Barge-in: stop TTS if it's playing
    if (isSpeaking) {
      stopTTS();
    }

    // Try to accumulate into complete sentences
    const completeSentence = accumulator.add(text);
    if (completeSentence) {
      handleTranslate(completeSentence);
    } else {
      // Set a flush timer
      if (flushTimerRef.current) clearTimeout(flushTimerRef.current);
      flushTimerRef.current = window.setTimeout(() => {
        const flushed = accumulator.flush();
        if (flushed) handleTranslate(flushed);
      }, 600);
    }
  }, [handleTranslate, isSpeaking, stopTTS]);

  const { isListening, interimTranscript, finalTranscript, error: asrError, isSupported, startListening, stopListening, resetTranscript } = useASR(handleFinalResult);

  // Audio visualizer
  const { canvasRef, startVisualizer, stopVisualizer } = useAudioVisualizer();

  // Handle mic button click
  const toggleListening = useCallback(() => {
    if (isListening) {
      stopListening();
      stopVisualizer();
      setAppState('idle');
      // Flush any remaining text
      const flushed = accumulator.flush();
      if (flushed) handleTranslate(flushed);
    } else {
      startListening();
      resetTranscript();
      accumulator.reset();
      startVisualizer();
      setAppState('listening');
      setTranslatedText('');
      setProcessedText('');
      setLatency(null);
      setStageStatuses({
        pre_process: 'idle',
        translate: 'idle',
        post_process: 'idle',
      });
    }
  }, [isListening, startListening, stopListening, startVisualizer, stopVisualizer, handleTranslate, resetTranscript]);

  // Handle WebSocket status updates
  useEffect(() => {
    if (!lastStatus) return;
    setStageStatuses(prev => ({
      ...prev,
      [lastStatus.stage]: lastStatus.status,
    }));
  }, [lastStatus]);

  // Handle translation results
  useEffect(() => {
    if (!lastResult) return;

    setTranslatedText(lastResult.translated_text);
    setCurrentEngine(lastResult.engine);
    setIsCached(lastResult.cached);
    setLatency(lastResult.latency);

    // Update pipeline stages to completed
    setStageStatuses({
      pre_process: 'completed',
      translate: 'completed',
      post_process: 'completed',
    });

    // Add to context history
    contextRef.current = [
      ...contextRef.current.slice(-4),
      lastResult.translated_text,
    ];

    // Add to history
    const entry: HistoryEntry = {
      id: Date.now().toString(),
      originalText: lastResult.original_text,
      translatedText: lastResult.translated_text,
      engine: lastResult.engine,
      latency: lastResult.latency,
      timestamp: new Date(),
      cached: lastResult.cached,
    };
    setHistory(prev => [entry, ...prev].slice(0, 50));

    // Auto-speak if enabled
    if (settings.autoSpeak) {
      setAppState('speaking');
      speak(lastResult.translated_text);
    } else {
      setAppState(isListening ? 'listening' : 'idle');
    }
  }, [lastResult, settings.autoSpeak, speak, isListening]);

  // Reset to listening state after TTS finishes
  useEffect(() => {
    if (!isSpeaking && appState === 'speaking') {
      setAppState(isListening ? 'listening' : 'idle');
    }
  }, [isSpeaking, appState, isListening]);

  // Handle errors
  useEffect(() => {
    if (asrError || lastError) {
      setAppState('error');
      // Reset stages on error so it doesn't spin forever
      setStageStatuses({
        pre_process: 'error',
        translate: 'error',
        post_process: 'error',
      });
    }
  }, [asrError, lastError]);

  const handleReplay = useCallback((text: string) => {
    speak(text);
  }, [speak]);

  return (
    <div className="app">
      {/* Header */}
      <header className="app-header">
        <div className="header-left">
          <h1 className="app-title">
            <span className="title-icon">🌐</span>
            VoiceFlow
            <span className="title-suffix">EN → HI</span>
          </h1>
        </div>
        <div className="header-right">
          <StatusBadge
            isConnected={isConnected}
            isListening={isListening}
            isSpeaking={isSpeaking}
          />
          <SettingsPanel
            settings={settings}
            onUpdate={updateSettings}
            isOpen={settingsOpen}
            onToggle={() => setSettingsOpen(!settingsOpen)}
          />
        </div>
      </header>

      {/* Browser compatibility warning */}
      {!isSupported && (
        <div className="compat-warning">
          <p>⚠️ Speech Recognition is not supported in this browser. Please use <strong>Chrome</strong> or <strong>Edge</strong> for the full experience.</p>
        </div>
      )}

      {/* Main content */}
      <main className="app-main">
        {/* Audio Visualizer */}
        <AudioVisualizer canvasRef={canvasRef} isActive={isListening} />

        {/* Pipeline Visualization */}
        <Pipeline
          currentStage={lastStatus?.stage || null}
          stageStatuses={stageStatuses}
        />

        {/* Mic Button */}
        <div className="mic-section">
          <MicrophoneButton
            state={appState}
            onClick={toggleListening}
            disabled={!isSupported}
          />
        </div>

        {/* Panels */}
        <div className="panels-grid">
          <TranscriptPanel
            finalTranscript={finalTranscript}
            interimTranscript={interimTranscript}
            processedText={processedText}
          />
          <TranslationPanel
            translatedText={translatedText}
            engine={currentEngine}
            cached={isCached}
            isSpeaking={isSpeaking}
            onSpeak={() => speak(translatedText)}
            onStop={stopTTS}
          />
        </div>

        {/* Latency Monitor */}
        <LatencyMonitor latency={latency} />

        {/* Error display */}
        {(asrError || lastError) && (
          <div className="error-banner">
            <span>⚠️ {asrError || lastError}</span>
          </div>
        )}

        {/* History */}
        <HistoryLog entries={history} onReplay={handleReplay} />
      </main>

      {/* Footer */}
      <footer className="app-footer">
        <p>Built with Go + React • Real-time English → Hindi Translation Pipeline</p>
      </footer>
    </div>
  );
}

export default App;
