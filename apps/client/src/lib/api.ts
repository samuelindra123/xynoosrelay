// XYNOSS Relay — Frontend API Client (Go Backend REST v1)

export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/v1";

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
    return localStorage.getItem("xynoss_csrf") || null;
  }
  return null;
}

export function setCsrfToken(token: string | null) {
  inMemoryCsrfToken = token;
  if (typeof window !== "undefined") {
    if (token) {
      localStorage.setItem("xynoss_csrf", token);
    } else {
      localStorage.removeItem("xynoss_csrf");
    }
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

  constructor(
    message: string,
    code: string = "INTERNAL_ERROR",
    status: number = 500,
    details?: ApiErrorDetail[],
    requestId?: string
  ) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
    this.details = details;
    this.requestId = requestId;
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

  const res = await fetch(url, {
    ...options,
    method,
    headers,
    credentials: "include", // Required for HTTP-only session cookies
  });

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
        throw new ApiError(text || res.statusText, "HTTP_ERROR", res.status);
      }
      return text as unknown as T;
    }
  }

  if (!res.ok) {
    const errObj = json?.error || {};
    throw new ApiError(
      errObj.message || res.statusText || "Permintaan gagal",
      errObj.code || "REQUEST_FAILED",
      res.status,
      errObj.details,
      errObj.request_id
    );
  }

  // If response contains csrf_token, keep it synchronized
  if (json?.data?.csrf_token) {
    setCsrfToken(json.data.csrf_token);
  }

  return (json?.data !== undefined ? json.data : json) as T;
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
  username: string;
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
  device_name: string;
  device_type: string;
  ip_address: string;
  is_current: boolean;
  last_active_at: string;
  expires_at: string;
}

export interface MediaUploadIntent {
  media_id: string;
  upload_url: string;
  method: string;
  headers: Record<string, string>;
  expires_at: string;
}

export interface MediaAssetEntity {
  id: string;
  purpose: string;
  mime_type: string;
  size_bytes: number;
  status: string;
  public_url: string;
}

export interface UserSearchResult {
  id: string;
  display_name: string;
  username: string;
  avatar_url?: string | null;
  bio?: string;
  availability?: string;
  custom_status?: string | null;
}

export interface ChannelEntity {
  id: string;
  workspace_id: string;
  slug: string;
  name: string;
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
    messages(id: string) { return apiFetch<ChannelMessageEntity[]>(`/conversations/${encodeURIComponent(id)}/messages`); },
    send(id: string, text: string, requestKey: string) {
      return apiFetch<ChannelMessageEntity>(`/conversations/${encodeURIComponent(id)}/messages`, { method: "POST", headers: { "Idempotency-Key": requestKey }, body: JSON.stringify({ text }) });
    },
  },
  channelMessages: {
    list(channelId: string) {
      return apiFetch<ChannelMessageEntity[]>(`/channels/${encodeURIComponent(channelId)}/messages`);
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
      try {
        const res = await fetch(`${API_BASE_URL}/auth/google/start`, {
          method: "GET",
          redirect: "manual",
        });
        if (res.status === 400 || res.status === 503) {
          const json = await res.json().catch(() => null);
          return {
            configured: false,
            error:
              json?.error?.message ||
              "Google OAuth belum dikonfigurasi di server backend.",
          };
        }
        return { configured: true };
      } catch {
        return {
          configured: false,
          error: "Tidak dapat terhubung ke server backend API.",
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
        headers["If-Match"] = `"${revision}"`;
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
      return apiFetch<void>("/users/me/avatar", {
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
      return apiFetch<void>("/users/me/banner", {
        method: "DELETE",
      });
    },

    async getSessions() {
      return apiFetch<{ sessions: SessionEntity[] }>("/me/sessions");
    },

    async searchUsers(query: string = "") {
      return apiFetch<UserSearchResult[]>(
        `/users/search?q=${encodeURIComponent(query)}`
      );
    },

    async revokeSession(sessionId: string) {
      return apiFetch<void>(`/me/sessions/${sessionId}`, {
        method: "DELETE",
      });
    },
  },

  media: {
    async requestUpload(data: {
      purpose: "avatar" | "banner";
      file_name: string;
      mime_type: string;
      size_bytes: number;
    }) {
      return apiFetch<MediaUploadIntent>("/media/uploads", {
        method: "POST",
        body: JSON.stringify(data),
      });
    },

    async uploadBinary(uploadUrl: string, file: File | Blob) {
      const res = await fetch(uploadUrl, {
        method: "PUT",
        headers: {
          "Content-Type": file.type || "application/octet-stream",
        },
        body: file,
      });
      if (!res.ok) {
        throw new ApiError(
          "Gagal mengunggah biner media ke storage",
          "STORAGE_UPLOAD_FAILED",
          res.status
        );
      }
    },

    async finalize(mediaId: string) {
      return apiFetch<{ media: MediaAssetEntity }>(
        `/media/${mediaId}/finalize`,
        {
          method: "POST",
        }
      );
    },

    async uploadAndFinalize(file: File, purpose: "avatar" | "banner") {
      // 1. Request upload intent
      const intent = await this.requestUpload({
        purpose,
        file_name: file.name,
        mime_type: file.type,
        size_bytes: file.size,
      });

      // 2. Direct binary PUT
      await this.uploadBinary(intent.upload_url, file);

      // 3. Finalize on backend
      const result = await this.finalize(intent.media_id);
      return result.media;
    },
  },

  workspaces: {
    async list() {
      return apiFetch<WorkspaceDetailEntity[]>("/workspaces");
    },
    async get(id: string) {
      return apiFetch<WorkspaceDetailEntity>(`/workspaces/${id}`);
    },
    async create(data: { name: string; description?: string }) {
      return apiFetch<WorkspaceDetailEntity>("/workspaces", {
        method: "POST",
        body: JSON.stringify(data),
      });
    },
    async createChannel(workspaceId: string, data: { name: string }) {
      return apiFetch<ChannelEntity>(`/workspaces/${workspaceId}/channels`, {
        method: "POST",
        body: JSON.stringify(data),
      });
    },
    async createInvite(workspaceId: string, data?: { duration_hours?: number; max_uses?: number }) {
      return apiFetch<CreateInviteResponse>(`/workspaces/${workspaceId}/invites`, {
        method: "POST",
        body: JSON.stringify(data || {}),
      });
    },
    async listInvites(workspaceId: string) {
      return apiFetch<WorkspaceInviteEntity[]>(`/workspaces/${workspaceId}/invites`);
    },
    async revokeInvite(workspaceId: string, inviteId: string) {
      return apiFetch<void>(`/workspaces/${workspaceId}/invites/${inviteId}`, {
        method: "DELETE",
      });
    },
  },

  invites: {
    async getPreview(code: string) {
      return apiFetch<InvitePreviewEntity>(`/invites/${code}`);
    },
    async accept(code: string) {
      return apiFetch<WorkspaceDetailEntity>(`/invites/${code}/accept`, {
        method: "POST",
      });
    },
  },
};
