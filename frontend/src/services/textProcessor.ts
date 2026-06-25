// Text processor — handles filler removal, NER protection, and sentence boundary detection

// Common English filler words/phrases to strip before translation
const FILLER_PATTERNS = [
  /\b(uh+|um+|hmm+|er+|ah+)\b/gi,
  /\b(you know)\b/gi,
  /\b(i mean)\b/gi,
  /\b(like,?\s)/gi,
  /\b(basically,?\s)/gi,
  /\b(actually,?\s)/gi,
  /\b(literally,?\s)/gi,
  /\b(so,?\s(?:yeah|like))\b/gi,
];

// Named entities that should be preserved as-is (not translated)
const PROTECTED_ENTITIES = [
  'Google Meet', 'Google', 'Zoom', 'WhatsApp', 'Slack', 'Teams',
  'Microsoft', 'Apple', 'Amazon', 'Netflix', 'Spotify',
  'Kubernetes', 'Docker', 'Asterisk', 'GitHub', 'GitLab',
  'React', 'Angular', 'Vue', 'Node.js', 'Python', 'Go',
  'FastAPI', 'Django', 'Flask', 'Express',
  'AWS', 'GCP', 'Azure', 'Redis', 'MongoDB', 'PostgreSQL',
  'ChatGPT', 'Gemini', 'Claude', 'OpenAI', 'Anthropic',
  'LinkedIn', 'Twitter', 'Instagram', 'Facebook', 'YouTube',
  'iPhone', 'Android', 'Windows', 'Linux', 'macOS',
];

/**
 * Remove filler words from the transcript.
 */
export function removeFillers(text: string): string {
  let cleaned = text;
  for (const pattern of FILLER_PATTERNS) {
    cleaned = cleaned.replace(pattern, ' ');
  }
  // Normalize whitespace
  return cleaned.replace(/\s+/g, ' ').trim();
}

/**
 * Check if the text appears to be a complete sentence/thought.
 * Returns true if it ends with punctuation or exceeds a minimum length.
 */
export function isCompleteSentence(text: string): boolean {
  const trimmed = text.trim();
  if (!trimmed) return false;

  // Ends with sentence-ending punctuation
  if (/[.!?]$/.test(trimmed)) return true;

  // Long enough to be a meaningful phrase (6+ words)
  const wordCount = trimmed.split(/\s+/).length;
  if (wordCount >= 6) return true;

  return false;
}

/**
 * Accumulate transcript fragments until we have a complete sentence.
 */
export class SentenceAccumulator {
  private buffer: string = '';
  private lastUpdateTime: number = 0;
  private pauseThresholdMs: number;

  constructor(pauseThresholdMs = 3000) {
    this.pauseThresholdMs = pauseThresholdMs;
  }

  /**
   * Add text to the buffer. Returns the complete sentence if ready, null otherwise.
   */
  add(text: string): string | null {
    this.buffer = text;
    this.lastUpdateTime = Date.now();

    const cleaned = removeFillers(this.buffer);
    if (isCompleteSentence(cleaned) && cleaned.length > 3) {
      const result = cleaned;
      this.buffer = '';
      return result;
    }

    return null;
  }

  /**
   * Force flush the buffer (e.g., after a long pause).
   */
  flush(): string | null {
    if (this.buffer.trim().length > 3) {
      const result = removeFillers(this.buffer);
      this.buffer = '';
      return result;
    }
    this.buffer = '';
    return null;
  }

  /**
   * Check if enough time has passed since the last update to force a flush.
   */
  shouldFlush(): boolean {
    if (!this.buffer) return false;
    return Date.now() - this.lastUpdateTime >= this.pauseThresholdMs;
  }

  /**
   * Get the current buffer contents.
   */
  getBuffer(): string {
    return this.buffer;
  }

  /**
   * Reset the buffer.
   */
  reset(): void {
    this.buffer = '';
    this.lastUpdateTime = 0;
  }
}

/**
 * Get the list of protected entity names.
 */
export function getProtectedEntities(): string[] {
  return [...PROTECTED_ENTITIES];
}
