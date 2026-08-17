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
