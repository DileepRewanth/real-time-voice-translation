interface TranscriptPanelProps {
  finalTranscript: string;
  interimTranscript: string;
  processedText: string;
}

export default function TranscriptPanel({
  finalTranscript,
  interimTranscript,
  processedText,
}: TranscriptPanelProps) {
  const hasContent = finalTranscript || interimTranscript;

  return (
    <div className="panel transcript-panel">
      <div className="panel-header">
        <div className="panel-icon">🎙️</div>
        <h3>English Transcription</h3>
        <span className="lang-badge">EN</span>
      </div>
      <div className="panel-content">
        {!hasContent && (
          <p className="placeholder-text">
            Start speaking to see your words appear here...
          </p>
        )}
        {finalTranscript && (
          <p className="transcript-text final">{finalTranscript}</p>
        )}
        {interimTranscript && (
          <p className="transcript-text interim">{interimTranscript}</p>
        )}
        {processedText && processedText !== finalTranscript && (
          <div className="processed-text">
            <span className="processed-label">Cleaned:</span>
            <p>{processedText}</p>
          </div>
        )}
      </div>
    </div>
  );
}
