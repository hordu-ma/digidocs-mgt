// Shared API error helpers.

type ApiErrorShape = {
  response?: { data?: { message?: unknown; code?: unknown } };
};

/**
 * Extracts a human-readable message from an axios error, falling back to the
 * provided default when the backend did not supply one. Centralizes the
 * `err.response?.data?.message` access used across views.
 */
export function extractError(err: unknown, fallback: string): string {
  const message = (err as ApiErrorShape)?.response?.data?.message;
  return typeof message === "string" && message !== "" ? message : fallback;
}
