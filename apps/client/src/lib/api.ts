// XYNOSS Relay — Frontend API Client (Go Backend REST v1)
import { safeApiError, type SafeApiErrorKind } from "./safe-api-error";
import { resolveApiBaseURL } from "./api-url";

export const API_BASE_URL = resolveApiBaseURL(process.env.NEXT_PUBLIC_API_URL, process.env.NODE_ENV || "production");

let inMemoryCsrfToken: string | null = null;
let inMemorySessionToken: string | null = null;

export function getSessionToken(): string | null {
  if (typeof window !== "undefined") {
    try { localStorage.removeItem("xynoss_session_token"); } catch { /* Storage may be unavailable. */ }
  }
  return inMemorySessionToken;
}

export function setSessionToken(token: string | null) {
  inMemorySessionToken = token;
  if (typeof window !== "undefined") {
    try { localStorage.removeItem("xynoss_session_token"); } catch { /* Storage may be unavailable. */ }
  }
}

export function getCsrfToken(): string | null {
  if (inMemoryCsrfToken) return inMemoryCsrfToken;
  if (typeof window !== "undefined") {
    try { return localStorage.getItem("xynoss_csrf") || null; } catch { return null; }
  }
  return null;
}

export function setCsrfToken(token: string | null) {
  inMemoryCsrfToken = token;
  if (typeof window !== "undefined") {
    try {
      if (token) localStorage.setItem("xynoss_csrf", token);
      else localStorage.removeItem("xynoss_csrf");
    } catch { /* Keep the in-memory token for this session. */ }
  }
}

export interface ApiErrorDetail {
  field?: string;
  code: string;
  message: string;
}

export class ApiError extends Error {
  code: string;
  details?: ApiErrorDetail[];
  status: number;
  requestId?: string;
  kind: SafeApiErrorKind;

  constructor(
    message: string,
    code: string = "INTERNAL_ERROR",
    status: number = 500,
    details?: ApiErrorDetail[],
    requestId?: string,
    kind?: SafeApiErrorKind
  ) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
    this.details = details;
    this.requestId = requestId;
    this.kind = kind || safeApiError(status).kind;
  }
}

export interface ApiResponse<T> {
  data: T;
  meta?: Record<string, unknown>;
  request_id?: string;
}

export async function apiFetch<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  if (!API_BASE_URL) {
    const safe = safeApiError(0);
    throw new ApiError(safe.message, "SERVICE_UNAVAILABLE", 0, undefined, undefined, safe.kind);
  }
  const url = `${API_BASE_URL}${path.startsWith("/") ? path : `/${path}`}`;
  const method = (options.method || "GET").toUpperCase();

  const headers = new Headers(options.headers || {});
  headers.set("Accept", "application/json");

  if (
    options.body &&
    typeof options.body === "string" &&
    !headers.has("Content-Type")
  ) {
    headers.set("Content-Type", "application/json");
  }

  const isMutating = ["POST", "PUT", "PATCH", "DELETE"].includes(method);
  if (isMutating) {
    const csrf = getCsrfToken();
    if (csrf && !headers.has("X-CSRF-Token")) {
      headers.set("X-CSRF-Token", csrf);
    }
  }

  const token = getSessionToken();
  if (token && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  let res: Response;
  try {
    res = await fetch(url, {
      ...options,
      method,
      headers,
      signal: options.signal || AbortSignal.timeout(30000),
      credentials: "include", // Required for HTTP-only session cookies
    });
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") throw error;
    const safe = safeApiError(0);
    throw new ApiError(safe.message, "SERVICE_UNAVAILABLE", 0, undefined, undefined, safe.kind);
  }

  if (res.status === 204) {
    return {} as T;
  }

  let json: { error?: { message?: string; code?: string; details?: ApiErrorDetail[]; request_id?: string }; data?: T & { csrf_token?: string } } | null = null;
  const text = await res.text();
  if (text) {
    try {
      json = JSON.parse(text);
    } catch {
      // Non-JSON response
      if (!res.ok) {
        const safe = safeApiError(res.status);
        throw new ApiError(safe.message, "REQUEST_FAILED", res.status, undefined, undefined, safe.kind);
      }
      throw new ApiError("Respons layanan tidak dapat dibaca. Coba lagi.", "INVALID_RESPONSE", 502);
    }
  }

  if (!res.ok) {
    const errObj = json?.error || {};
    const safe = safeApiError(res.status);
    throw new ApiError(
      safe.message,
      res.status >= 500 ? "SERVICE_ERROR" : errObj.code || "REQUEST_FAILED",
      res.status,
      errObj.details,
      errObj.request_id,
      safe.kind
    );
  }

  // If response contains csrf_token, keep it synchronized
  if (json?.data?.csrf_token) {
    setCsrfToken(json.data.csrf_token);
  }

  if (!json || !("data" in json)) throw new ApiError("Respons layanan tidak lengkap. Coba lagi.", "INVALID_RESPONSE", 502);
  return json.data as T;
}

// Data Entities
export interface UserEntity {
  id: string;
  display_name?: string;
  username?: string;
  avatar_url?: string | null;
  system_role?: string;
  created_at?: string;
}

export interface ProfileEntity {
  display_name: string;
  username?: string;
  bio?: string;
  avatar_media_id?: string | null;
  banner_media_id?: string | null;
  avatar_url?: string | null;
  banner_url?: string | null;
  availability?: string;
  custom_status?: string;
  revision: number;
}

export interface IdentityEntity {
  provider: string;
  identifier: string;
}

export interface SessionEntity {
  id: string;
  label?: string;
  platform_hint?: string;
  is_current: boolean;
  last_seen_at: string;
  idle_expires_at: string;
  absolute_expires_at: string;
}

export interface MediaUploadIntent {
  media_id: string;
  upload_url: string;
  method: string;
  required_headers: Record<string, string>;
  expires_at: string;
}

export interface MediaAssetEntity {
  id: string;
  declared_type: string;
  bytes: number;
  status: string;
  url?: string;
}

export interface UserSearchResult {
  id: string;
  display_name: string;
  username: string;
  avatar_url?: string | null;
  banner_url?: string | null;
  bio?: string;
  availability?: string;
  custom_status?: string | null;
}

export interface ChannelEntity {
  id: string;
  workspace_id: string;
  slug: string;
  name: string;
  visibility: "public" | "private";
  position: number;
  created_at: string;
}

export interface ChannelMessageEntity {
  id: string;
  sequence: number;
  author_id: string;
  author: string;
  username: string;
  text: string;
  created_at: string;
  revision?: number;
}

export interface DirectConversationEntity {
  id: string;
  user_id: string;
  username: string;
  display_name: string;
}

export interface WorkspaceMemberEntity {
  id: string;
  workspace_id: string;
  user_id: string;
  role: "Owner" | "Admin" | "Moderator" | "Member";
  title?: string;
  joined_at: string;
  display_name: string;
  username: string;
  avatar_url?: string;
}

export interface WorkspaceDetailEntity {
  id: string;
  name: string;
  slug: string;
  description: string;
  owner_user_id: string;
  created_at: string;
  channels: ChannelEntity[];
  members: WorkspaceMemberEntity[];
}

export interface InvitePreviewEntity {
  code: string;
  workspace_id: string;
  workspace_name: string;
  workspace_description: string;
  member_count: number;
  inviter_name: string;
  inviter_avatar?: string;
  invite_url: string;
}

export interface CreateInviteResponse {
  code: string;
  invite_url: string;
  expires_at: string;
}

export interface WorkspaceInviteEntity {
  id: string;
  workspace_id: string;
  created_by: string;
  expires_at: string;
  max_uses: number;
  uses: number;
  revoked_at?: string | null;
  created_at: string;
  status: "active" | "expired" | "exhausted" | "revoked";
}

// API Service Client
export const api = {
  conversations: {
    list() { return apiFetch<DirectConversationEntity[]>("/conversations"); },
    open(userId: string) {
      return apiFetch<DirectConversationEntity>("/conversations/direct", { method: "POST", body: JSON.stringify({ user_id: userId }) });
    },
    messages(id: string, before?: number) { return apiFetch<ChannelMessageEntity[]>(`/conversations/${encodeURIComponent(id)}/messages${before ? `?before_sequence=${before}` : ""}`); },
    read(id: string, sequence?: number) { return apiFetch<{ last_read_sequence: number }>(`/conversations/${encodeURIComponent(id)}/read`, sequence === undefined ? {} : { method: "POST", body: JSON.stringify({ last_read_sequence: sequence }) }); },
    edit(id: string, sequence: number, text: string, revision: number) { return apiFetch<{sequence:number;text:string;revision:number}>(`/conversations/${encodeURIComponent(id)}/messages/${sequence}`, { method:"PATCH", body:JSON.stringify({text,revision}) }); },
    delete(id: string, sequence: number) { return apiFetch<void>(`/conversations/${encodeURIComponent(id)}/messages/${sequence}`, { method:"DELETE" }); },
    react(id: string, sequence: number, active: boolean) { return apiFetch<void>(`/conversations/${encodeURIComponent(id)}/messages/${sequence}/reactions/heart`, { method:active?"PUT":"DELETE" }); },
    send(id: string, text: string, requestKey: string) {
      return apiFetch<ChannelMessageEntity>(`/conversations/${encodeURIComponent(id)}/messages`, { method: "POST", headers: { "Idempotency-Key": requestKey }, body: JSON.stringify({ text }) });
    },
  },
  channelMessages: {
    async privateAccess(channelId: string) {
      return apiFetch<{ user_ids: string[] }>(`/channels/${encodeURIComponent(channelId)}/access`);
    },
    async grantPrivateAccess(channelId: string, userId: string) {
      return apiFetch<void>(`/channels/${encodeURIComponent(channelId)}/access`, { method:"PUT", body:JSON.stringify({user_id:userId}) });
    },
    async revokePrivateAccess(channelId: string, userId: string) {
      return apiFetch<void>(`/channels/${encodeURIComponent(channelId)}/access/${encodeURIComponent(userId)}`, { method:"DELETE" });
    },
    read(id: string, sequence?: number) { return apiFetch<{ last_read_sequence: number }>(`/channels/${encodeURIComponent(id)}/read`, sequence === undefined ? {} : { method: "POST", body: JSON.stringify({ last_read_sequence: sequence }) }); },
    edit(id: string, sequence: number, text: string, revision: number) { return apiFetch<{sequence:number;text:string;revision:number}>(`/channels/${encodeURIComponent(id)}/messages/${sequence}`, { method:"PATCH", body:JSON.stringify({text,revision}) }); },
    delete(id: string, sequence: number) { return apiFetch<void>(`/channels/${encodeURIComponent(id)}/messages/${sequence}`, { method:"DELETE" }); },
    react(id: string, sequence: number, active: boolean) { return apiFetch<void>(`/channels/${encodeURIComponent(id)}/messages/${sequence}/reactions/heart`, { method:active?"PUT":"DELETE" }); },
    list(channelId: string, before?: number) {
      return apiFetch<ChannelMessageEntity[]>(`/channels/${encodeURIComponent(channelId)}/messages${before ? `?before_sequence=${before}` : ""}`);
    },
    send(channelId: string, text: string, requestKey: string) {
      return apiFetch<ChannelMessageEntity>(`/channels/${encodeURIComponent(channelId)}/messages`, {
        method: "POST",
        headers: { "Idempotency-Key": requestKey },
        body: JSON.stringify({ text }),
      });
    },
  },
  health: {
    async isOnline(): Promise<boolean> {
      if (!API_BASE_URL) return false;
      try {
        const res = await fetch(`${API_BASE_URL}/health`, {
          method: "GET",
          headers: { Accept: "application/json" },
        });
        return res.ok;
      } catch {
        return false;
      }
    },
  },

  auth: {
    async requestOtp(email: string) {
      return apiFetch<{
        challenge_id: string;
        retry_after_seconds: number;
      }>("/auth/otp/challenges", {
        method: "POST",
        body: JSON.stringify({ email }),
      });
    },

    async checkGoogleConfigured(): Promise<{
      configured: boolean;
      error?: string;
    }> {
      if (!API_BASE_URL) return { configured: false, error: safeApiError(0).message };
      try {
        const res = await fetch(`${API_BASE_URL}/auth/google/start`, {
          method: "GET",
          redirect: "manual",
          signal: AbortSignal.timeout(4000),
        });
        if (!res.ok && res.type !== "opaqueredirect" && res.status !== 0) {
          return {
            configured: false,
            error: "Masuk dengan Google belum tersedia saat ini. Coba lagi sebentar atau gunakan email.",
          };
        }
        return { configured: true };
      } catch {
        return {
          configured: false,
          error: "Kami belum bisa memulai masuk dengan Google. Coba lagi sebentar atau gunakan email.",
        };
      }
    },

    async verifyOtp(challengeId: string, code: string) {
      const data = await apiFetch<{
        user: UserEntity;
        onboarding_required: boolean;
        csrf_token: string;
      }>("/auth/otp/verifications", {
        method: "POST",
        body: JSON.stringify({ challenge_id: challengeId, code }),
      });
      if (data.csrf_token) {
        setCsrfToken(data.csrf_token);
      }
      return data;
    },

    getGoogleStartUrl(redirectUri?: string) {
      if (!API_BASE_URL) {
        const safe = safeApiError(0);
        throw new ApiError(safe.message, "SERVICE_UNAVAILABLE", 0, undefined, undefined, safe.kind);
      }
      const params = new URLSearchParams();
      if (redirectUri) {
        params.set("return_path", redirectUri);
      }
      const qs = params.toString();
      return `${API_BASE_URL}/auth/google/start${qs ? `?${qs}` : ""}`;
    },

    async getSession() {
      const data = await apiFetch<{
        user: UserEntity;
        onboarding_required: boolean;
        csrf_token: string;
      }>("/auth/session");
      if (data.csrf_token) {
        setCsrfToken(data.csrf_token);
      }
      return data;
    },

    async logout() {
      try {
        await apiFetch<void>("/auth/logout", {
          method: "POST",
        });
      } finally {
        setCsrfToken(null);
        setSessionToken(null);
      }
    },
  },

  user: {
    async getMe() {
      const data = await apiFetch<{
        id: string;
        username?: string | null;
        onboarding_required: boolean;
        profile: Omit<ProfileEntity, "username">;
      }>("/me");
      return {
        user: { id: data.id, username: data.username || undefined, display_name: data.profile?.display_name } as UserEntity,
        profile: { ...data.profile, username: data.username || "" } as ProfileEntity,
        identities: [] as IdentityEntity[],
        onboarding_completed: !data.onboarding_required,
      };
    },

    async completeOnboarding(data: { display_name: string; username?: string }) {
      return apiFetch<{
        user: UserEntity;
        onboarding_required: boolean;
      }>("/me/onboarding/complete", {
        method: "POST",
        body: JSON.stringify(data),
      });
    },

    async getProfile() {
      return apiFetch<ProfileEntity>("/me/profile");
    },

    async updateProfile(
      fields: Partial<Omit<ProfileEntity, "revision">>,
      revision?: number
    ) {
      const headers: Record<string, string> = {};
      if (revision !== undefined) {
        headers["If-Match"] = String(revision);
      }
      return apiFetch<ProfileEntity>("/me/profile", {
        method: "PATCH",
        headers,
        body: JSON.stringify(fields),
      });
    },

    async setAvatar(mediaId: string) {
      return apiFetch<ProfileEntity>("/users/me/avatar", {
        method: "POST",
        body: JSON.stringify({ media_id: mediaId }),
      });
    },

    async deleteAvatar() {
      return apiFetch<ProfileEntity>("/users/me/avatar", {
        method: "DELETE",
      });
    },

    async setBanner(mediaId: string) {
      return apiFetch<ProfileEntity>("/users/me/banner", {
        method: "POST",
        body: JSON.stringify({ media_id: mediaId }),
      });
    },

    async deleteBanner() {
      return apiFetch<ProfileEntity>("/users/me/banner", {
        method: "DELETE",
      });
    },

    async getSessions() {
      return apiFetch<SessionEntity[]>("/me/sessions");
    },

    async searchUsers(query: string = "") {
      return apiFetch<UserSearchResult[]>(
        `/users/search?q=${encodeURIComponent(query)}`,
        { cache: "no-store" }
      );
    },

    async revokeSession(sessionId: string) {
      return apiFetch<void>(`/me/sessions/${encodeURIComponent(sessionId)}`, {
        method: "DELETE",
      });
    },
  },

  media: {
    async requestUpload(data: {
      purpose: "avatar" | "banner";
      content_type: string;
      size_bytes: number;
    }) {
      return apiFetch<MediaUploadIntent>("/media/uploads", {
        method: "POST",
        body: JSON.stringify(data),
      });
    },

    async uploadBinary(intent: MediaUploadIntent, file: File | Blob, onProgress?: (percent: number) => void) {
      if (Date.parse(intent.expires_at) <= Date.now()) throw new ApiError("Tautan unggah kedaluwarsa. Mulai unggahan baru.", "UPLOAD_EXPIRED", 410);
      // Signed storage requests use only the headers returned by the backend.
      // Never forward session credentials or CSRF tokens to object storage.
      return new Promise<void>((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        xhr.open(intent.method, intent.upload_url);
        xhr.timeout = 120000;
        for (const [name, value] of Object.entries(intent.required_headers || {})) xhr.setRequestHeader(name, value);
        xhr.upload.onprogress = event => { if (event.lengthComputable) onProgress?.(Math.round(event.loaded / event.total * 100)); };
        const failed = () => reject(new ApiError("Unggahan belum berhasil. Periksa koneksi lalu coba lagi.", "STORAGE_UPLOAD_FAILED", 0));
        xhr.onerror = failed;
        xhr.ontimeout = failed;
        xhr.onabort = failed;
        xhr.onload = () => xhr.status >= 200 && xhr.status < 300 ? resolve() : reject(new ApiError(xhr.status === 403 ? "Tautan unggah ditolak atau kedaluwarsa. Mulai unggahan baru." : "Unggahan ditolak penyimpanan. Coba lagi.", "STORAGE_UPLOAD_FAILED", xhr.status));
        xhr.send(file);
      });
    },

    get(mediaId: string) {
      return apiFetch<MediaAssetEntity>(`/media/${encodeURIComponent(mediaId)}`);
    },

    delete(mediaId: string) {
      return apiFetch<void>(`/media/${encodeURIComponent(mediaId)}`, { method: "DELETE" });
    },

    async finalize(mediaId: string) {
      return apiFetch<MediaAssetEntity>(
        `/media/${encodeURIComponent(mediaId)}/finalize`,
        {
          method: "POST",
        }
      );
    },

    async uploadAndFinalize(file: File, purpose: "avatar" | "banner") {
      // 1. Request upload intent
      const intent = await this.requestUpload({
        purpose,
        content_type: file.type,
        size_bytes: file.size,
      });

      // 2. Direct binary PUT
      await this.uploadBinary(intent, file);

      // 3. Finalize on backend
      const result = await this.finalize(intent.media_id);
      return result;
    },
  },

  workspaces: {
    async list() {
      return apiFetch<WorkspaceDetailEntity[]>("/workspaces");
    },
    async get(id: string) {
      return apiFetch<WorkspaceDetailEntity>(`/workspaces/${encodeURIComponent(id)}`);
    },
    async create(data: { name: string; description?: string }) {
      return apiFetch<WorkspaceDetailEntity>("/workspaces", {
        method: "POST",
        body: JSON.stringify(data),
      });
    },
    async createChannel(workspaceId: string, data: { name: string; visibility?: "public" | "private" }) {
      return apiFetch<ChannelEntity>(`/workspaces/${encodeURIComponent(workspaceId)}/channels`, {
        method: "POST",
        body: JSON.stringify(data),
      });
    },
    async createInvite(workspaceId: string, data?: { duration_hours?: number; max_uses?: number }) {
      return apiFetch<CreateInviteResponse>(`/workspaces/${encodeURIComponent(workspaceId)}/invites`, {
        method: "POST",
        body: JSON.stringify(data || {}),
      });
    },
    async listInvites(workspaceId: string) {
      return apiFetch<WorkspaceInviteEntity[]>(`/workspaces/${encodeURIComponent(workspaceId)}/invites`);
    },
    async revokeInvite(workspaceId: string, inviteId: string) {
      return apiFetch<void>(`/workspaces/${encodeURIComponent(workspaceId)}/invites/${encodeURIComponent(inviteId)}`, {
        method: "DELETE",
      });
    },
  },

  invites: {
    async getPreview(code: string) {
      return apiFetch<InvitePreviewEntity>(`/invites/${encodeURIComponent(code)}`);
    },
    async accept(code: string) {
      return apiFetch<WorkspaceDetailEntity>(`/invites/${encodeURIComponent(code)}/accept`, {
        method: "POST",
      });
    },
  },
};
