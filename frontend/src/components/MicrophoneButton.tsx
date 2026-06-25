import type { AppState } from '../types';

interface MicrophoneButtonProps {
  state: AppState;
  onClick: () => void;
  disabled?: boolean;
}

export default function MicrophoneButton({ state, onClick, disabled }: MicrophoneButtonProps) {
  const isActive = state === 'listening';
  const isProcessing = state === 'processing';

  return (
    <button
      className={`mic-button ${isActive ? 'active' : ''} ${isProcessing ? 'processing' : ''}`}
      onClick={onClick}
      disabled={disabled}
      aria-label={isActive ? 'Stop listening' : 'Start listening'}
    >
      <div className="mic-ripple" />
      <div className="mic-ripple delay-1" />
      <div className="mic-ripple delay-2" />
      <div className="mic-icon">
        {isActive ? (
          <svg viewBox="0 0 24 24" fill="currentColor" width="32" height="32">
            <rect x="6" y="6" width="12" height="12" rx="2" />
          </svg>
        ) : (
          <svg viewBox="0 0 24 24" fill="currentColor" width="32" height="32">
            <path d="M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3z" />
            <path d="M17 11c0 2.76-2.24 5-5 5s-5-2.24-5-5H5c0 3.53 2.61 6.43 6 6.92V21h2v-3.08c3.39-.49 6-3.39 6-6.92h-2z" />
          </svg>
        )}
      </div>
      <span className="mic-label">
        {state === 'idle' && 'Tap to speak'}
        {state === 'listening' && 'Listening...'}
        {state === 'processing' && 'Processing...'}
        {state === 'speaking' && 'Speaking...'}
        {state === 'error' && 'Tap to retry'}
      </span>
    </button>
  );
}
