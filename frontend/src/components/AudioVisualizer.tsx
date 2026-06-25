interface AudioVisualizerProps {
  canvasRef: React.RefObject<HTMLCanvasElement | null>;
  isActive: boolean;
}

export default function AudioVisualizer({ canvasRef, isActive }: AudioVisualizerProps) {
  return (
    <div className={`visualizer-container ${isActive ? 'active' : ''}`}>
      <canvas
        ref={canvasRef}
        className="visualizer-canvas"
      />
      {!isActive && (
        <div className="visualizer-idle">
          <div className="idle-bars">
            {Array.from({ length: 20 }).map((_, i) => (
              <div key={i} className="idle-bar" style={{ animationDelay: `${i * 0.05}s` }} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
