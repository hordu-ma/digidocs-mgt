// Shared formatting helpers.

/**
 * Formats a byte count into a human-readable size string (B / KB / MB / GB).
 * Returns "-" for missing input; a genuine 0-byte file renders as "0 B".
 */
export function formatFileSize(bytes?: number | null): string {
  if (bytes === undefined || bytes === null || Number.isNaN(bytes)) return "-";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

function pad(n: number): string {
  return n < 10 ? `0${n}` : `${n}`;
}

function toDate(value?: string | null): Date | null {
  if (!value) return null;
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? null : d;
}

/**
 * Unified timestamp rendering: "2026-11-10 14:05" in local time. Always
 * includes the year so dates never read ambiguously across a year boundary.
 * Missing or invalid input renders as `fallback`.
 */
export function formatDateTime(value?: string | null, fallback = "-"): string {
  const d = toDate(value);
  if (!d) return fallback;
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** Date-only variant: "2026-11-10". */
export function formatDate(value?: string | null, fallback = "-"): string {
  const d = toDate(value);
  if (!d) return fallback;
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}
