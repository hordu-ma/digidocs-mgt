import { describe, expect, it } from "vitest";

import { resolveAuthRedirect } from "./index";

function storage(values: Record<string, string | null>) {
  return {
    getItem(key: string) {
      return values[key] ?? null;
    },
  };
}

describe("route auth guard", () => {
  it("redirects protected routes to login when token is missing", () => {
    expect(resolveAuthRedirect({ requiresAuth: true }, storage({}))).toEqual({ name: "login" });
  });

  it("allows protected routes when token exists", () => {
    expect(
      resolveAuthRedirect(
        { requiresAuth: true },
        storage({
          access_token: "token-1",
          role: "member",
        }),
      ),
    ).toBeUndefined();
  });

  it("redirects non-admin users away from admin routes", () => {
    expect(
      resolveAuthRedirect(
        { requiresAuth: true, requiresAdmin: true },
        storage({
          access_token: "token-1",
          role: "member",
        }),
      ),
    ).toEqual({ name: "dashboard" });
  });

  it("allows admin routes for admin users", () => {
    expect(
      resolveAuthRedirect(
        { requiresAuth: true, requiresAdmin: true },
        storage({
          access_token: "token-1",
          role: "admin",
        }),
      ),
    ).toBeUndefined();
  });
});
