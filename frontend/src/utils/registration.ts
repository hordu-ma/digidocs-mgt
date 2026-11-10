import axios from "axios";
import { ref } from "vue";

// SELF_REGISTRATION mode from GET /system/info: open | invite_only | closed.
// Defaults to "open" when the call fails, matching the backend default.
export const registrationMode = ref<"open" | "invite_only" | "closed">("open");
let loaded = false;

export async function loadRegistrationMode() {
  if (loaded) return;
  loaded = true;
  try {
    const res = await axios.get("/api/v1/system/info");
    const mode = res.data?.data?.self_registration;
    if (mode === "invite_only" || mode === "closed") registrationMode.value = mode;
  } catch {
    // keep default
  }
}

/** Invite token from a "/join/<token>" redirect, if any. */
export function inviteTokenFromRedirect(redirect: unknown): string {
  if (typeof redirect !== "string") return "";
  const m = redirect.match(/^\/join\/([^/?#]+)/);
  return m ? m[1] : "";
}
