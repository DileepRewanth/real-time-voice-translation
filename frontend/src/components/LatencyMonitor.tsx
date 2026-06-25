import type { LatencyBreakdown } from '../types';

interface LatencyMonitorProps {
  latency: LatencyBreakdown | null;
}

export default function LatencyMonitor({ latency }: LatencyMonitorProps) {
  if (!latency) return null;

  const stages = [
    { label: 'Pre-Process', value: latency.pre_process_ms, color: '#f093fb' },
    { label: 'Translate', value: latency.translate_ms, color: '#4facfe' },
    { label: 'Post-Process', value: latency.post_process_ms, color: '#43e97b' },
  ];

  const maxMs = Math.max(latency.total_ms, 1);

  return (
    <div className="latency-monitor">
      <div className="latency-header">
        <span className="latency-icon">⚡</span>
        <span className="latency-total">{latency.total_ms}ms total</span>
      </div>
      <div className="latency-bars">
        {stages.map((stage) => (
          <div key={stage.label} className="latency-bar-row">
            <span className="latency-bar-label">{stage.label}</span>
            <div className="latency-bar-track">
              <div
                className="latency-bar-fill"
                style={{
                  width: `${Math.max((stage.value / maxMs) * 100, 2)}%`,
                  backgroundColor: stage.color,
                }}
              />
            </div>
            <span className="latency-bar-value">{stage.value}ms</span>
          </div>
        ))}
      </div>
    </div>
  );
}
