import { useRef, useCallback, useEffect } from 'react';

interface UseAudioVisualizerReturn {
  canvasRef: React.RefObject<HTMLCanvasElement | null>;
  startVisualizer: () => Promise<void>;
  stopVisualizer: () => void;
}

export function useAudioVisualizer(): UseAudioVisualizerReturn {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const animationRef = useRef<number | null>(null);
  const analyserRef = useRef<AnalyserNode | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const audioContextRef = useRef<AudioContext | null>(null);

  const draw = useCallback(() => {
    const canvas = canvasRef.current;
    const analyser = analyserRef.current;
    if (!canvas || !analyser) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const bufferLength = analyser.frequencyBinCount;
    const dataArray = new Uint8Array(bufferLength);
    analyser.getByteFrequencyData(dataArray);

    const width = canvas.width;
    const height = canvas.height;

    // Clear with transparent background
    ctx.clearRect(0, 0, width, height);

    // Draw frequency bars
    const barCount = 64;
    const barWidth = width / barCount;
    const step = Math.floor(bufferLength / barCount);

    for (let i = 0; i < barCount; i++) {
      const value = dataArray[i * step];
      const barHeight = (value / 255) * height * 0.85;

      // Gradient colors matching the app theme
      const hue = 250 + (i / barCount) * 120; // Purple to green
      const saturation = 70 + (value / 255) * 30;
      const lightness = 50 + (value / 255) * 20;

      ctx.fillStyle = `hsla(${hue}, ${saturation}%, ${lightness}%, 0.8)`;

      // Center-aligned bars
      const x = i * barWidth + 1;
      const y = (height - barHeight) / 2;

      ctx.beginPath();
      ctx.roundRect(x, y, barWidth - 2, barHeight, 3);
      ctx.fill();
    }

    animationRef.current = requestAnimationFrame(draw);
  }, []);

  const startVisualizer = useCallback(async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      streamRef.current = stream;

      const audioContext = new AudioContext();
      audioContextRef.current = audioContext;

      const source = audioContext.createMediaStreamSource(stream);
      const analyser = audioContext.createAnalyser();
      analyser.fftSize = 256;
      analyser.smoothingTimeConstant = 0.8;

      source.connect(analyser);
      analyserRef.current = analyser;

      // Set canvas resolution
      if (canvasRef.current) {
        const rect = canvasRef.current.getBoundingClientRect();
        canvasRef.current.width = rect.width * window.devicePixelRatio;
        canvasRef.current.height = rect.height * window.devicePixelRatio;
      }

      draw();
    } catch (err) {
      console.error('Failed to start audio visualizer:', err);
    }
  }, [draw]);

  const stopVisualizer = useCallback(() => {
    if (animationRef.current) {
      cancelAnimationFrame(animationRef.current);
      animationRef.current = null;
    }
    if (streamRef.current) {
      streamRef.current.getTracks().forEach(track => track.stop());
      streamRef.current = null;
    }
    if (audioContextRef.current) {
      audioContextRef.current.close();
      audioContextRef.current = null;
    }
    analyserRef.current = null;

    // Clear canvas
    if (canvasRef.current) {
      const ctx = canvasRef.current.getContext('2d');
      if (ctx) ctx.clearRect(0, 0, canvasRef.current.width, canvasRef.current.height);
    }
  }, []);

  // Cleanup on unmount
  useEffect(() => {
    return () => stopVisualizer();
  }, [stopVisualizer]);

  return { canvasRef, startVisualizer, stopVisualizer };
}
