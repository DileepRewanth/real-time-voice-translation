import type { PipelineStage } from '../types';

interface PipelineProps {
  currentStage: PipelineStage | null;
  stageStatuses: Record<PipelineStage, 'idle' | 'processing' | 'completed' | 'error'>;
}

const stages: { key: PipelineStage; icon: string; label: string }[] = [
  { key: 'pre_process', icon: '🧾', label: 'Pre-Process' },
  { key: 'translate', icon: '🌐', label: 'Translate' },
  { key: 'post_process', icon: '✨', label: 'Post-Process' },
];

export default function Pipeline({ stageStatuses }: PipelineProps) {
  return (
    <div className="pipeline">
      <div className="pipeline-stages">
        {stages.map((stage, index) => {
          const status = stageStatuses[stage.key];
          return (
            <div key={stage.key} className="pipeline-stage-wrapper">
              <div className={`pipeline-stage ${status}`}>
                <div className="pipeline-icon">{stage.icon}</div>
                <div className="pipeline-label">{stage.label}</div>
                <div className={`pipeline-indicator ${status}`}>
                  {status === 'processing' && <div className="spinner" />}
                  {status === 'completed' && '✓'}
                  {status === 'error' && '✗'}
                </div>
              </div>
              {index < stages.length - 1 && (
                <div className={`pipeline-connector ${status === 'completed' ? 'active' : ''}`}>
                  <div className="connector-line" />
                  <div className="connector-arrow">→</div>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
