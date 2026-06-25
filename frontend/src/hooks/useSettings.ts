import { useState, useCallback, useEffect } from 'react';
import type { Settings } from '../types';

const STORAGE_KEY = 'voice-translator-settings';

const DEFAULT_SETTINGS: Settings = {
  engine: 'mymemory',
  geminiApiKey: '',
  tonePreference: '',
  ttsRate: 1,
  ttsPitch: 1,
  autoSpeak: true,
};

export function useSettings(): [Settings, (updates: Partial<Settings>) => void] {
  const [settings, setSettings] = useState<Settings>(() => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored) {
        return { ...DEFAULT_SETTINGS, ...JSON.parse(stored) };
      }
    } catch {
      // Ignore parse errors
    }
    return DEFAULT_SETTINGS;
  });

  // Persist to localStorage on change
  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
    } catch {
      // Ignore storage errors
    }
  }, [settings]);

  const updateSettings = useCallback((updates: Partial<Settings>) => {
    setSettings(prev => ({ ...prev, ...updates }));
  }, []);

  return [settings, updateSettings];
}
