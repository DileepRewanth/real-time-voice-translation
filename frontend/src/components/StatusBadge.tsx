interface StatusBadgeProps {
  isConnected: boolean;
  isListening: boolean;
  isSpeaking: boolean;
}

export default function StatusBadge({ isConnected, isListening, isSpeaking }: StatusBadgeProps) {
  return (
    <div className="status-badges">
      <div className={`status-badge ${isConnected ? 'connected' : 'disconnected'}`}>
        <div className="status-dot" />
        <span>{isConnected ? 'Connected' : 'Disconnected'}</span>
      </div>
      {isListening && (
        <div className="status-badge listening">
          <div className="status-dot pulse" />
          <span>Listening</span>
        </div>
      )}
      {isSpeaking && (
        <div className="status-badge speaking">
          <div className="status-dot pulse" />
          <span>Speaking</span>
        </div>
      )}
    </div>
  );
}
