import type { HistoryEntry } from '../types';

interface HistoryLogProps {
  entries: HistoryEntry[];
  onReplay: (text: string) => void;
}

export default function HistoryLog({ entries, onReplay }: HistoryLogProps) {
  if (entries.length === 0) {
    return (
      <div className="panel history-panel">
        <div className="panel-header">
          <div className="panel-icon">📜</div>
          <h3>Translation History</h3>
        </div>
        <div className="panel-content">
          <p className="placeholder-text">
            Your translation history will appear here...
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="panel history-panel">
      <div className="panel-header">
        <div className="panel-icon">📜</div>
        <h3>Translation History</h3>
        <span className="history-count">{entries.length}</span>
      </div>
      <div className="panel-content history-list">
        {entries.map((entry) => (
          <div key={entry.id} className="history-entry">
            <div className="history-texts">
              <p className="history-original">{entry.originalText}</p>
              <p className="history-translated">{entry.translatedText}</p>
            </div>
            <div className="history-meta">
              <span className={`engine-badge small ${entry.engine}`}>
                {entry.engine === 'gemini' ? '✦' : '◆'}
              </span>
              {entry.cached && <span className="cache-badge small">⚡</span>}
              <span className="latency-badge">{entry.latency.total_ms}ms</span>
              <button
                className="replay-btn"
                onClick={() => onReplay(entry.translatedText)}
                title="Replay Hindi audio"
              >
                🔊
              </button>
              <span className="history-time">
                {entry.timestamp.toLocaleTimeString()}
              </span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
