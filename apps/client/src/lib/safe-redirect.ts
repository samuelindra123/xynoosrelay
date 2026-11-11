/** Only application and invitation paths are valid post-authentication targets. */
export function safeAuthRedirect(value: string | null | undefined): string {
  if (!value || value.length > 2048 || !value.startsWith("/") || value.startsWith("//") || /[\\\u0000-\u001f\u007f]/.test(value)) return "/app";
  try {
    const base = "https://relay.invalid";
    const url = new URL(value, base);
    if (url.origin !== base || /%2f|%5c/i.test(url.pathname)) return "/app";
    if (url.pathname !== "/app" && !url.pathname.startsWith("/app/") && !url.pathname.startsWith("/invite/")) return "/app";
    return url.pathname + url.search + url.hash;
  } catch { return "/app"; }
}
