import { fetch as tauriFetch, type ClientOptions } from "@tauri-apps/plugin-http";
import { getVersion } from "@tauri-apps/api/app";

// Cache the user agent string
let userAgentCache: string | null = null;

/**
 * Get the user agent string for HTTP requests.
 * Format: "cloudreve-desktop/{version}"
 */
async function getUserAgent(): Promise<string> {
  if (userAgentCache) {
    return userAgentCache;
  }
  const version = await getVersion();
  userAgentCache = `cloudreve-desktop/${version}`;
  return userAgentCache;
}

/**
 * Wrapper around Tauri's fetch that automatically adds the User-Agent header.
 * Use this instead of importing fetch directly from @tauri-apps/plugin-http.
 */
export async function fetch(
  input: URL | Request | string,
  init?: RequestInit & ClientOptions
): Promise<Response> {
  const userAgent = await getUserAgent();

  const headers = new Headers(init?.headers);
  if (!headers.has("User-Agent")) {
    headers.set("User-Agent", userAgent);
  }

  return tauriFetch(input, {
    ...init,
    headers,
  });
}
