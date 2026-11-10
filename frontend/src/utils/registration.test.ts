import { describe, expect, it } from "vitest";

import { inviteTokenFromRedirect } from "./registration";

describe("inviteTokenFromRedirect", () => {
  it("extracts the token from a join redirect", () => {
    expect(inviteTokenFromRedirect("/join/abc123")).toBe("abc123");
    expect(inviteTokenFromRedirect("/join/abc123?x=1")).toBe("abc123");
  });
  it("ignores other redirects", () => {
    expect(inviteTokenFromRedirect("/dashboard")).toBe("");
    expect(inviteTokenFromRedirect(undefined)).toBe("");
  });
});
