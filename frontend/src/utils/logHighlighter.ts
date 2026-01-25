/**
 * Log syntax highlighter for terminal output
 * Provides colorful highlighting for logs without ANSI codes
 */

// Escape HTML special characters
const escapeHtml = (text: string): string => {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
};

// Strip ANSI escape codes from text
const stripAnsi = (text: string): string => {
  // eslint-disable-next-line no-control-regex
  return text.replace(/\x1b\[[0-9;]*m/g, '');
};

// Highlight rules - order matters! More specific rules should come first
const highlightRules: Array<{
  pattern: RegExp;
  className: string;
  isLink?: boolean;
}> = [
  // Python Traceback header
  { pattern: /Traceback \(most recent call last\):/g, className: 'log-error' },
  // Python exception types
  {
    pattern: /\b(\w*Error|\w*Exception)(?=:|\s|$)/g,
    className: 'log-error-type',
  },
  // Log levels - error
  { pattern: /\b(ERROR|CRITICAL|FATAL)\b/g, className: 'log-error' },
  // Log levels - warning
  { pattern: /\b(WARNING|WARN)\b/g, className: 'log-warning' },
  // Log levels - info
  { pattern: /\b(INFO)\b/g, className: 'log-info' },
  // Log levels - debug
  { pattern: /\b(DEBUG)\b/g, className: 'log-debug' },
  // Log levels - success
  { pattern: /\b(SUCCESS|OK|PASSED)\b/g, className: 'log-success' },
  // URL (must be before path matching)
  { pattern: /https?:\/\/[^\s<>"']+/g, className: 'log-url', isLink: true },
  // Timestamp with date
  {
    pattern: /\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d{3})?/g,
    className: 'log-time',
  },
  // Timestamp without date
  { pattern: /\b\d{2}:\d{2}:\d{2}(?:\.\d{3})?\b/g, className: 'log-time' },
  // Windows file path
  { pattern: /[A-Za-z]:\\[^\s:*?"<>|]+/g, className: 'log-path' },
  // Python boolean and None
  { pattern: /\bTrue\b/g, className: 'log-bool-true' },
  { pattern: /\bFalse\b/g, className: 'log-bool-false' },
  { pattern: /\bNone\b/g, className: 'log-none' },
  // Double-quoted strings
  { pattern: /"[^"\\]*(?:\\.[^"\\]*)*"/g, className: 'log-string' },
  // Single-quoted strings
  { pattern: /'[^'\\]*(?:\\.[^'\\]*)*'/g, className: 'log-string' },
];

interface Segment {
  text: string;
  className?: string;
  isLink?: boolean;
  start: number;
  end: number;
}

/**
 * Highlight a single line of log text
 */
export const highlightLog = (text: string): string => {
  if (!text) return '';

  // First strip any ANSI codes
  text = stripAnsi(text);

  // Find all matches and their positions
  const matches: Segment[] = [];

  for (const rule of highlightRules) {
    // Reset regex lastIndex
    rule.pattern.lastIndex = 0;
    let match;
    while ((match = rule.pattern.exec(text)) !== null) {
      const start = match.index;
      const end = start + match[0].length;

      // Check if this region overlaps with any existing match
      const overlaps = matches.some(
        (m) =>
          (start >= m.start && start < m.end) ||
          (end > m.start && end <= m.end) ||
          (start <= m.start && end >= m.end),
      );

      if (!overlaps) {
        matches.push({
          text: match[0],
          className: rule.className,
          isLink: rule.isLink ?? false,
          start,
          end,
        });
      }
    }
  }

  // Sort by position
  matches.sort((a, b) => a.start - b.start);

  // Build result
  const parts: string[] = [];
  let lastEnd = 0;

  for (const m of matches) {
    // Add text before this match
    if (m.start > lastEnd) {
      parts.push(escapeHtml(text.slice(lastEnd, m.start)));
    }

    // Add the highlighted match
    const escaped = escapeHtml(m.text);
    if (m.isLink) {
      parts.push(
        `<a href="${escapeHtml(m.text)}" target="_blank" class="${m.className}">${escaped}</a>`,
      );
    } else {
      parts.push(`<span class="${m.className}">${escaped}</span>`);
    }

    lastEnd = m.end;
  }

  // Add remaining text
  if (lastEnd < text.length) {
    parts.push(escapeHtml(text.slice(lastEnd)));
  }

  return parts.join('');
};

/**
 * Highlight multiple lines of log text
 */
export const highlightLogs = (logs: string[]): string[] => {
  return logs.map(highlightLog);
};
