export function resolveApiBaseURL(configured: string | undefined, environment: string): string {
  const value = configured?.trim();
  if (value) {
    try {
      const url = new URL(value);
      if (!(url.protocol === "http:" || url.protocol === "https:") || url.username || url.password) return "";
      const hostname = url.hostname.toLowerCase().replace(/^\[|\]$/g, "");
      const loopback = hostname === "localhost" || hostname.endsWith(".localhost") || hostname === "::1" || hostname === "0.0.0.0" || /^127(?:\.\d{1,3}){3}$/.test(hostname);
      if (environment === "production" && loopback) return "";
      return url.toString().replace(/\/+$/, "");
    } catch {
      return "";
    }
  }
  return "";
}
