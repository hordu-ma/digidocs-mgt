import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, it } from "vitest";

import { useAuthStore } from "./auth";

describe("auth store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  it("persists login identity to state and localStorage", () => {
    const auth = useAuthStore();

    auth.login({
      token: "token-1",
      id: "user-1",
      username: "zhangsan",
      displayName: "张三",
      role: "admin",
    });

    expect(auth.token).toBe("token-1");
    expect(auth.userId).toBe("user-1");
    expect(auth.username).toBe("zhangsan");
    expect(auth.displayName).toBe("张三");
    expect(auth.role).toBe("admin");
    expect(localStorage.getItem("access_token")).toBe("token-1");
    expect(localStorage.getItem("user_id")).toBe("user-1");
    expect(localStorage.getItem("role")).toBe("admin");
  });

  it("updates profile fields without touching the access token", () => {
    const auth = useAuthStore();
    auth.login({
      token: "token-1",
      id: "user-1",
      username: "zhangsan",
      displayName: "张三",
      role: "member",
    });

    auth.updateProfile({
      username: "zhangsan",
      displayName: "张老师",
      role: "project_lead",
    });

    expect(auth.token).toBe("token-1");
    expect(localStorage.getItem("access_token")).toBe("token-1");
    expect(auth.displayName).toBe("张老师");
    expect(auth.role).toBe("project_lead");
    expect(localStorage.getItem("display_name")).toBe("张老师");
  });

  it("clears all persisted identity on logout", () => {
    const auth = useAuthStore();
    auth.login({
      token: "token-1",
      id: "user-1",
      username: "zhangsan",
      displayName: "张三",
      role: "admin",
    });

    auth.logout();

    expect(auth.token).toBeNull();
    expect(auth.userId).toBe("");
    expect(auth.username).toBe("");
    expect(auth.displayName).toBe("");
    expect(auth.role).toBe("");
    expect(localStorage.getItem("access_token")).toBeNull();
    expect(localStorage.getItem("user_id")).toBeNull();
    expect(localStorage.getItem("role")).toBeNull();
  });
});
