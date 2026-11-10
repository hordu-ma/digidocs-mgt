import { describe, expect, it } from "vitest";

import { formatDate, formatDateTime, formatFileSize } from "./format";

describe("format helpers", () => {
  it("renders timestamps with the year and zero padding", () => {
    const local = new Date(2026, 0, 5, 9, 7).toISOString();
    expect(formatDateTime(local)).toBe("2026-01-05 09:07");
    expect(formatDate(local)).toBe("2026-01-05");
  });

  it("uses the fallback for missing or invalid input", () => {
    expect(formatDateTime(undefined)).toBe("-");
    expect(formatDateTime("", "暂无")).toBe("暂无");
    expect(formatDateTime("not-a-date", "暂无")).toBe("暂无");
  });

  it("keeps a real zero-byte size", () => {
    expect(formatFileSize(0)).toBe("0 B");
  });
});
