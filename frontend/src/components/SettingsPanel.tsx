import type { Settings } from '../types';

interface SettingsPanelProps {
  settings: Settings;
  onUpdate: (updates: Partial<Settings>) => void;
  isOpen: boolean;
  onToggle: () => void;
}

export default function SettingsPanel({ settings, onUpdate, isOpen, onToggle }: SettingsPanelProps) {
  return (
    <>
      <button className="settings-toggle" onClick={onToggle} title="Settings">
        ⚙️
      </button>
      {isOpen && (
        <div className="settings-overlay" onClick={onToggle}>
          <div className="settings-panel" onClick={(e) => e.stopPropagation()}>
            <div className="settings-header">
              <h3>⚙️ Settings</h3>
              <button className="settings-close" onClick={onToggle}>✕</button>
            </div>

            <div className="settings-body">
              {/* Translation Engine */}
              <div className="setting-group">
                <label className="setting-label">Translation Engine</label>
                <div className="setting-options">
                  <button
                    className={`option-btn ${settings.engine === 'mymemory' ? 'active' : ''}`}
                    onClick={() => onUpdate({ engine: 'mymemory' })}
                  >
                    <span className="option-icon">◆</span>
                    <div>
                      <div className="option-name">MyMemory</div>
                      <div className="option-desc">Free, no API key needed</div>
                    </div>
                  </button>
                  <button
                    className={`option-btn ${settings.engine === 'gemini' ? 'active' : ''}`}
                    onClick={() => onUpdate({ engine: 'gemini' })}
                  >
                    <span className="option-icon">✦</span>
                    <div>
                      <div className="option-name">Gemini</div>
                      <div className="option-desc">AI-powered, context-aware</div>
                    </div>
                  </button>
                </div>
              </div>

              {/* Gemini API Key */}
              {settings.engine === 'gemini' && (
                <div className="setting-group">
                  <label className="setting-label">Gemini API Key</label>
                  <input
                    type="password"
                    className="setting-input"
                    value={settings.geminiApiKey}
                    onChange={(e) => onUpdate({ geminiApiKey: e.target.value })}
                    placeholder="Enter your Gemini API key..."
                  />
                  <p className="setting-hint">
                    Get a free key from{' '}
                    <a href="https://aistudio.google.com/" target="_blank" rel="noreferrer">
                      Google AI Studio
                    </a>
                  </p>
                </div>
              )}

              {/* Tone Preference */}
              <div className="setting-group">
                <label className="setting-label">Hindi Tone</label>
                <div className="setting-options horizontal">
                  <button
                    className={`option-btn small ${settings.tonePreference === '' ? 'active' : ''}`}
                    onClick={() => onUpdate({ tonePreference: '' })}
                  >
                    Auto
                  </button>
                  <button
                    className={`option-btn small ${settings.tonePreference === 'formal' ? 'active' : ''}`}
                    onClick={() => onUpdate({ tonePreference: 'formal' })}
                  >
                    Formal (आप)
                  </button>
                  <button
                    className={`option-btn small ${settings.tonePreference === 'casual' ? 'active' : ''}`}
                    onClick={() => onUpdate({ tonePreference: 'casual' })}
                  >
                    Casual (तुम)
                  </button>
                </div>
              </div>

              {/* TTS Settings */}
              <div className="setting-group">
                <label className="setting-label">Speech Rate: {settings.ttsRate.toFixed(1)}</label>
                <input
                  type="range"
                  min="0.5"
                  max="2"
                  step="0.1"
                  value={settings.ttsRate}
                  onChange={(e) => onUpdate({ ttsRate: parseFloat(e.target.value) })}
                  className="setting-slider"
                />
              </div>

              <div className="setting-group">
                <label className="setting-label">Speech Pitch: {settings.ttsPitch.toFixed(1)}</label>
                <input
                  type="range"
                  min="0.5"
                  max="2"
                  step="0.1"
                  value={settings.ttsPitch}
                  onChange={(e) => onUpdate({ ttsPitch: parseFloat(e.target.value) })}
                  className="setting-slider"
                />
              </div>

              {/* Auto-speak */}
              <div className="setting-group">
                <label className="setting-toggle-label">
                  <input
                    type="checkbox"
                    checked={settings.autoSpeak}
                    onChange={(e) => onUpdate({ autoSpeak: e.target.checked })}
                  />
                  <span className="toggle-slider" />
                  Auto-speak translations
                </label>
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
