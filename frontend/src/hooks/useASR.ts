import { useState, useCallback, useRef, useEffect } from 'react';

interface ASRState {
  isListening: boolean;
  interimTranscript: string;
  finalTranscript: string;
  error: string | null;
  isSupported: boolean;
}

interface UseASRReturn extends ASRState {
  startListening: () => void;
  stopListening: () => void;
  resetTranscript: () => void;
}

export function useASR(onFinalResult?: (text: string) => void): UseASRReturn {
  const [isListening, setIsListening] = useState(false);
  const [interimTranscript, setInterimTranscript] = useState('');
  const [finalTranscript, setFinalTranscript] = useState('');
  const [error, setError] = useState<string | null>(null);

  const recognitionRef = useRef<SpeechRecognition | null>(null);
  const onFinalResultRef = useRef(onFinalResult);
  onFinalResultRef.current = onFinalResult;

  const isSupported = typeof window !== 'undefined' &&
    ('SpeechRecognition' in window || 'webkitSpeechRecognition' in window);

  useEffect(() => {
    if (!isSupported) return;

    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    const recognition = new SpeechRecognition();

    recognition.continuous = true;
    recognition.interimResults = true;
    recognition.lang = 'en-US';
    recognition.maxAlternatives = 1;

    recognition.onresult = (event: SpeechRecognitionEvent) => {
      let interim = '';
      let final = '';

      for (let i = event.resultIndex; i < event.results.length; i++) {
        const transcript = event.results[i][0].transcript;
        if (event.results[i].isFinal) {
          final += transcript;
        } else {
          interim += transcript;
        }
      }

      if (interim) {
        setInterimTranscript(interim);
      }

      if (final) {
        setFinalTranscript(final);
        setInterimTranscript('');
        onFinalResultRef.current?.(final);
      }
    };

    recognition.onerror = (event: SpeechRecognitionErrorEvent) => {
      // Ignore no-speech errors to allow continuous listening without disconnecting
      if (event.error === 'no-speech') {
        return;
      }

      const errorMessages: Record<string, string> = {
        'not-allowed': 'Microphone access denied. Please allow microphone permissions.',
        'network': 'Network error. Please check your connection.',
        'audio-capture': 'No microphone found. Please connect a microphone.',
        'aborted': 'Recognition was aborted.',
      };
      setError(errorMessages[event.error] || `Speech recognition error: ${event.error}`);
      setIsListening(false);
    };

    recognition.onend = () => {
      // Auto-restart if we're still supposed to be listening
      if (recognitionRef.current && isListening) {
        try {
          recognition.start();
        } catch {
          // Ignore — may already be running
        }
      }
    };

    recognitionRef.current = recognition;

    return () => {
      recognition.abort();
      recognitionRef.current = null;
    };
  }, [isSupported]);

  // Re-attach onend handler when isListening changes
  useEffect(() => {
    const recognition = recognitionRef.current;
    if (!recognition) return;

    recognition.onend = () => {
      if (isListening) {
        try {
          recognition.start();
        } catch {
          // Ignore
        }
      } else {
        setIsListening(false);
      }
    };
  }, [isListening]);

  const startListening = useCallback(() => {
    if (!recognitionRef.current) return;
    setError(null);
    setInterimTranscript('');
    setIsListening(true);
    try {
      recognitionRef.current.start();
    } catch {
      // Ignore if already started, but we ensure React state is updated above
    }
  }, []);

  const stopListening = useCallback(() => {
    if (!recognitionRef.current) return;
    setIsListening(false);
    recognitionRef.current.stop();
    setInterimTranscript('');
  }, []);

  const resetTranscript = useCallback(() => {
    setFinalTranscript('');
    setInterimTranscript('');
  }, []);

  return {
    isListening,
    interimTranscript,
    finalTranscript,
    error,
    isSupported,
    startListening,
    stopListening,
    resetTranscript,
  };
}
