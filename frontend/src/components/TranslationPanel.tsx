interface TranslationPanelProps {
  translatedText: string;
  engine: string;
  cached: boolean;
  isSpeaking: boolean;
  onSpeak: () => void;
  onStop: () => void;
}

export default function TranslationPanel({
  translatedText,
  engine,
  cached,
  isSpeaking,
  onSpeak,
  onStop,
}: TranslationPanelProps) {
  return (
    <div className="panel translation-panel">
      <div className="panel-header">
        <div className="panel-icon">🗣️</div>
        <h3>Hindi Translation</h3>
        <span className="lang-badge hindi">HI</span>
      </div>
      <div className="panel-content">
        {!translatedText && (
          <p className="placeholder-text">
            Hindi translation will appear here...
          </p>
        )}
        {translatedText && (
          <>
            <p className="translation-text">{translatedText}</p>
            <div className="translation-meta">
              <span className={`engine-badge ${engine}`}>
                {engine === 'gemini' ? '✦ Gemini' : '◆ MyMemory'}
              </span>
              {cached && <span className="cache-badge">⚡ Cached</span>}
              <button
                className={`speak-btn ${isSpeaking ? 'speaking' : ''}`}
                onClick={isSpeaking ? onStop : onSpeak}
              >
                {isSpeaking ? '⏹ Stop' : '▶ Speak'}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
