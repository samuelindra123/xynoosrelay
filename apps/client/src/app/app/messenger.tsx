"use client";

import { useCallback, useEffect, useRef, useState, useMemo, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import "./messenger.css";
import { unreadCount, canMarkRead } from "./read-state";
import { pendingChannelRequest } from "./channel-retry";
import {
  api,
  type SessionEntity,
  type UserSearchResult,
  type WorkspaceDetailEntity,
  type InvitePreviewEntity,
  type ChannelMessageEntity,
  type WorkspaceInviteEntity,
  ApiError,
} from "@/lib/api";
import { signOutEverywhere } from "@/lib/auth/sign-out";
import {
  MessageSquare,
  Users,
  Hash,
  Search,
  Pin,
  Info,
  Settings,
  Plus,
  Copy,
  Check,
  X,
  ArrowUpRight,
  SendHorizontal,
  Camera,
  Image as ImageIcon,
  Shield,
  Crown,
  Sparkles,
  Smile,
  LogOut,
  Key,
  UserPlus,
  Menu,
  ChevronLeft,
  ChevronRight,
  Loader2,
} from "lucide-react";

// --- TYPES ---
export type Msg = {
  id: number;
  authorId?: string;
  author: string;
  authorUsername?: string;
  authorAvatar?: string;
  text: string;
  time: string;
  reaction?: string;
  reply?: string;
  pin?: boolean;
};

export type DMContact = {
  id: string;
  conversationId?: string;
  name: string;
  username: string;
  avatar?: string;
  status?: string;
  customStatus?: string;
  bio?: string;
  banner?: string;
};

export type UserProfile = {
  name: string;
  username: string;
  bio?: string;
  status?: string;
  custom?: string;
  avatar?: string;
  banner?: string;
  revision?: number;
  role?: string;
};

// --- LUXURY COLOR PALETTES & BANNER PRESETS ---
const PALETTES = [
  { bg: "linear-gradient(135deg, #40564a 0%, #253c31 100%)", text: "#e7f0d9" },
  { bg: "linear-gradient(135deg, #b65837 0%, #d67d56 100%)", text: "#ffffff" },
  { bg: "linear-gradient(135deg, #354d5e 0%, #243b49 100%)", text: "#b7d9ec" },
  { bg: "linear-gradient(135deg, #70583b 0%, #8a6a43 100%)", text: "#faf0db" },
  { bg: "linear-gradient(135deg, #5b3d68 0%, #3e2646 100%)", text: "#f6e8fa" },
  { bg: "linear-gradient(135deg, #2f5446 0%, #1f3b30 100%)", text: "#d7ebd9" },
  { bg: "linear-gradient(135deg, #874332 0%, #5e2a1d 100%)", text: "#fce9e4" },
  { bg: "linear-gradient(135deg, #3c4c5e 0%, #253342 100%)", text: "#deecfc" },
];

function getPalette(name: string) {
  let hash = 0;
  for (let i = 0; i < (name || "").length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  return PALETTES[Math.abs(hash) % PALETTES.length];
}

const BANNER_PRESETS = [
  "linear-gradient(135deg, #1e2922 0%, #283b32 50%, #3a5245 100%)",
  "linear-gradient(135deg, #2c322b 0%, #444a42 50%, #565e53 100%)",
  "linear-gradient(135deg, #243b49 0%, #354d5e 50%, #48687e 100%)",
  "linear-gradient(135deg, #3f2d25 0%, #583c2e 50%, #6f4b39 100%)",
  "linear-gradient(135deg, #2b3a32 0%, #394d42 50%, #4a6456 100%)",
];

function getBanner(name: string) {
  let hash = 0;
  for (let i = 0; i < (name || "").length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  return BANNER_PRESETS[Math.abs(hash) % BANNER_PRESETS.length];
}

// --- LUXURY AVATAR COMPONENT ---
function LuxuryAvatar({
  name,
  src,
  size = 40,
  status,
  className = "",
  onClick,
}: {
  name: string;
  src?: string;
  size?: number;
  status?: string;
  className?: string;
  onClick?: () => void;
}) {
  const [failedSrc, setFailedSrc] = useState<string | undefined>();
  const imgErr = Boolean(src && src === failedSrc);
  const palette = useMemo(() => getPalette(name || "?"), [name]);
  const initial = useMemo(() => {
    const clean = (name || "?").trim();
    return clean.slice(0, Math.min(2, clean.length)).toUpperCase();
  }, [name]);

  const statusColor =
    status === "online" || status === "Tersedia"
      ? "#22c55e"
      : status === "busy" || status === "Sibuk"
      ? "#ef4444"
      : status === "away" || status === "Jauh"
      ? "#f59e0b"
      : undefined;

  return (
    <div
      className={`luxury-avatar-wrap ${className}`}
      style={{ width: size, height: size }}
      onClick={onClick}
    >
      <div
        className="luxury-avatar"
        style={{
          width: size,
          height: size,
          background: !src || imgErr ? palette.bg : "transparent",
          color: palette.text,
          fontSize: Math.max(12, Math.floor(size * 0.38)),
        }}
      >
        {src && !imgErr ? (
          // User-supplied image URLs may be hosted outside the Next image allowlist.
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={src}
            alt={name}
            onError={() => setFailedSrc(src)}
            className="luxury-avatar-img"
          />
        ) : (
          <span>{initial}</span>
        )}
      </div>
      {statusColor && (
        <span
          className="luxury-avatar-badge"
          style={{
            backgroundColor: statusColor,
            width: Math.max(9, Math.floor(size * 0.28)),
            height: Math.max(9, Math.floor(size * 0.28)),
          }}
        />
      )}
    </div>
  );
}

// --- PROFILE CARD POPUP ---
function ProfileCard({
  user,
  onClose,
  onMessage,
}: {
  user: UserProfile;
  onClose: () => void;
  onMessage: (user: UserProfile) => void;
}) {
  const [copied, setCopied] = useState(false);
  const bannerBg = user.banner ? `url(${user.banner})` : getBanner(user.username || user.name);

  function copyId() {
    navigator.clipboard?.writeText(`@${user.username}`).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  return (
    <div className="discord-card-overlay" onClick={onClose}>
      <div className="discord-profile-card" onClick={(e) => e.stopPropagation()}>
        <div
          className="discord-card-banner"
          style={{
            backgroundImage: user.banner ? `url(${user.banner})` : undefined,
            background: !user.banner ? bannerBg : undefined,
          }}
        >
          <button className="discord-card-close" onClick={onClose} aria-label="Tutup profil">
            <X size={16} />
          </button>
        </div>
        <div className="discord-card-header">
          <div className="discord-avatar-container">
            <LuxuryAvatar
              name={user.name}
              src={user.avatar}
              size={76}
              status={user.status || "online"}
            />
          </div>
          <div className="discord-badges">
            {user.role === "Owner" ? (
              <span className="role-badge role-owner">
                <Crown size={12} /> Owner
              </span>
            ) : (
              <span className="role-badge role-member">
                <Shield size={12} /> Anggota
              </span>
            )}
            <span className="role-badge role-xynoss">
              <Sparkles size={12} /> XYNOSS
            </span>
          </div>
        </div>
        <div className="discord-card-content">
          <div className="discord-name-box">
            <h3 className="discord-display-name">{user.name}</h3>
            <span className="discord-username">@{user.username}</span>
          </div>

          {user.custom && (
            <div className="discord-custom-status">
              <MessageSquare size={13} style={{ flexShrink: 0, opacity: 0.8 }} />
              <span>{user.custom}</span>
            </div>
          )}

          <div className="discord-divider" />

          <div>
            <div className="discord-section-title">TENTANG SAYA</div>
            <p className="discord-bio-text">
              {user.bio || "Pengguna ini belum menambahkan catatan tentang dirinya."}
            </p>
          </div>

          <div className="discord-actions">
            <button
              className="discord-btn-primary"
              onClick={() => onMessage(user)}
            >
              <MessageSquare size={14} /> Kirim Pesan
            </button>
            <button className="discord-btn-secondary" onClick={copyId}>
              {copied ? (
                <>
                  <Check size={13} /> Tersalin!
                </>
              ) : (
                <>
                  <Copy size={13} /> Salin ID
                </>
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

// --- MAIN MESSENGER COMPONENT ---
export default function Messenger({ accountId }: { accountId: string }) {
  const router = useRouter();
  const storagePrefix = `xynoss_v3_${accountId}_`;

  // Profile of current logged-in user
  const [profile, setProfile] = useState<UserProfile>({
    name: "Kamu",
    username: "kamu",
    bio: "",
    status: "Tersedia",
    custom: "",
    avatar: "",
    banner: "",
    revision: 1,
  });

  // Persistent state using storage versioning (xynoss_v2_*)
  const [spaces, setSpaces] = useState<WorkspaceDetailEntity[]>([]);
  const [dmContacts, setDmContacts] = useState<DMContact[]>([]);
  const [, setMessages] = useState<Record<string, Msg[]>>({});
  const [channelMessages, setChannelMessages] = useState<Record<string, Msg[]>>({});
  const [sendingChannelMessage, setSendingChannelMessage] = useState(false);
  const [channelError, setChannelError] = useState("");
  const [readCursors, setReadCursors] = useState<Record<string, number>>({});
  const [storageReady, setStorageReady] = useState(false);
  const [theme, setTheme] = useState<"light" | "dark">("light");
  const [joiningWorkspace, setJoiningWorkspace] = useState(false);

  // Active navigation state
  const [space, setSpace] = useState("dm"); // "dm" or workspace id
  const [selected, setSelected] = useState(""); // username in dm, channel slug in workspace
  const [drafts, setDrafts] = useState<Record<string, string>>({});
  const [search, setSearch] = useState("");
  const [panel, setPanel] = useState("");
  const [dialog, setDialog] = useState("");
  const [picker, setPicker] = useState("");
  const [reply, setReply] = useState<Msg | null>(null);
  const [editing, setEditing] = useState<number | null>(null);

  // Invite & Join modals
  const [viewingUser, setViewingUser] = useState<UserProfile | null>(null);
  const [inviteWsTarget, setInviteWsTarget] = useState<WorkspaceDetailEntity | null>(null);
  const [inviteGeneratedLink, setInviteGeneratedLink] = useState("");
  const [inviteGeneratedCode, setInviteGeneratedCode] = useState("");
  const [loadingInvite, setLoadingInvite] = useState(false);
  const [workspaceInvites, setWorkspaceInvites] = useState<WorkspaceInviteEntity[]>([]);
  const [inviteDurationHours, setInviteDurationHours] = useState(168);
  const [inviteMaxUses, setInviteMaxUses] = useState(0);
  const [revokingInviteID, setRevokingInviteID] = useState("");
  const [joinWsOffer, setJoinWsOffer] = useState<InvitePreviewEntity | null>(null);
  const [manualJoinCode, setManualJoinCode] = useState("");

  // Search real users state
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<UserSearchResult[]>([]);
  const [searching, setSearching] = useState(false);
  const [copiedLink, setCopiedLink] = useState(false);

  // Workspace creation & channel creation state
  const [newName, setNewName] = useState("");
  const [newDesc, setNewDesc] = useState("");
  const [creatingWs, setCreatingWs] = useState(false);
  const [notice, setNotice] = useState("");
  const [uploadingField, setUploadingField] = useState<"avatar" | "banner" | null>(null);
  const [realSessions, setRealSessions] = useState<SessionEntity[]>([]);
  const [mobileList, setMobileList] = useState(false);
  const [railPinned, setRailPinned] = useState(false);
  const [isConnected, setIsConnected] = useState(false);
  const [savingProfile, setSavingProfile] = useState(false);
  const chatBodyRef = useRef<HTMLDivElement>(null);
  const channelMutationRef = useRef(0);
  const bannerInputRef = useRef<HTMLInputElement>(null);
  const avatarInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    try {
      const saved = localStorage.getItem("xynoss_app_theme");
      // Hydrate the browser-only preference after the server render.
      // eslint-disable-next-line react-hooks/set-state-in-effect
      if (saved === "dark" || saved === "light") setTheme(saved);
      else if (window.matchMedia("(prefers-color-scheme: dark)").matches) setTheme("dark");
    } catch { /* Appearance remains usable when storage is unavailable. */ }
  }, []);

  function changeTheme(next: "light" | "dark") {
    setTheme(next);
    try { localStorage.setItem("xynoss_app_theme", next); } catch { /* Keep the current selection for this session. */ }
  }

  // Data from earlier unscoped versions is deliberately left untouched: its
  // owner cannot be determined safely on a shared browser.
  useEffect(() => {
    try {
      const savedCursors = localStorage.getItem(`${storagePrefix}server_read_cursors`);
      if (savedCursors) {
        // Restore the account-scoped browser cursor when its storage key changes.
        // eslint-disable-next-line react-hooks/set-state-in-effect
        setReadCursors(JSON.parse(savedCursors));
      }
    } catch {
      setNotice("Beberapa preferensi tidak dapat dimuat. Periksa izin penyimpanan browser.");
    } finally { setStorageReady(true); }
  }, [storagePrefix]);

  // Save V2 changes to localStorage
  useEffect(() => {
    if (!storageReady) return;
    try { localStorage.setItem(`${storagePrefix}server_read_cursors`, JSON.stringify(readCursors)); } catch { /* Reading remains available without persistence. */ }
  }, [readCursors, storageReady, storagePrefix]);

  // 2. Load Real Workspaces from Database Backend (Starts at 0 workspaces for new users)
  const refreshWorkspaces = useCallback(async () => {
    try {
      const list = await api.workspaces.list();
      if (Array.isArray(list)) {
        setSpaces(list);
      }
    } catch {}
  }, []);

  // 3. Bootstrap user session & profile
  useEffect(() => {
    let active = true;

    async function bootstrapUser() {
      try {
        const data = await api.user.getMe();
        if (!active) return;
        setIsConnected(true);
        if (data && data.profile) {
          setProfile({
            name: data.profile.display_name || data.user.display_name || "Kamu",
            username: data.profile.username || data.user.username || "kamu",
            bio: data.profile.bio || "",
            status: data.profile.availability || "Tersedia",
            custom: data.profile.custom_status || "",
            avatar: data.profile.avatar_url || "",
            banner: data.profile.banner_url || "",
            revision: data.profile.revision || 1,
          });
        }
        refreshWorkspaces();
      } catch {
        if (active) setIsConnected(false);
      }
    }

    bootstrapUser();

    return () => {
      active = false;
    };
  }, [refreshWorkspaces]);

  // 4. Handle URL Invitation Query Parameters (?join=[code] or ?chat_with=@user or ?workspace=[id])
  useEffect(() => {
    if (typeof window === "undefined") return;
    const params = new URLSearchParams(window.location.search);
    const joinCode = params.get("join") || params.get("join_workspace");
    const chatWith = params.get("chat_with");
    const wsParam = params.get("workspace");

    if (wsParam) {
      // Apply and consume an external navigation request.
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setSpace(wsParam);
      window.history.replaceState({}, "", window.location.pathname);
    }

    if (joinCode) {
      api.invites
        .getPreview(joinCode)
        .then((prev) => {
          setJoinWsOffer(prev);
        })
        .catch(() => {
          setNotice("Kode undangan tidak ditemukan atau sudah kedaluwarsa.");
        });
      window.history.replaceState({}, "", window.location.pathname);
    }

    if (chatWith) {
      const cleanUsername = chatWith.replace(/^@/, "").trim();
      if (cleanUsername) {
        api.user
          .searchUsers(cleanUsername)
          .then(async (res) => {
            const found = res?.find(
              (u) =>
                u.username.toLowerCase() === cleanUsername.toLowerCase() ||
                u.id === cleanUsername
            );
            if (found) {
              const conversation = await api.conversations.open(found.id);
              const newContact: DMContact = {
                id: found.id,
                conversationId: conversation.id,
                username: found.username,
                name: found.display_name || found.username,
                avatar: found.avatar_url || "",
                status: found.availability || "online",
                customStatus: found.custom_status || "",
                bio: found.bio || "",
              };
              setDmContacts((prev) => {
                return [newContact, ...prev.filter((c) => c.id !== found.id)];
              });
              setSpace("dm");
              setSelected(found.username);
            }
          })
          .catch(() => setNotice("Percakapan dari tautan tidak dapat dibuka. Periksa koneksi dan nama pengguna."));
      }
      window.history.replaceState({}, "", window.location.pathname);
    }
  }, []);

  // 4b. Ensure default channel is selected when workspace is active but channel is unset
  useEffect(() => {
    if (space !== "dm" && !selected && spaces.length > 0) {
      const activeWs = spaces.find((w) => w.id === space);
      if (activeWs && activeWs.channels && activeWs.channels.length > 0) {
        // Select the first channel when externally loaded workspace data arrives.
        // eslint-disable-next-line react-hooks/set-state-in-effect
        setSelected(activeWs.channels[0].slug);
      }
    }
  }, [space, spaces, selected]);

  // 5. Handle Live User Search
  useEffect(() => {
    let active = true;
    if (!searchQuery.trim()) {
      // Clear stale results as the query is cleared.
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setSearchResults([]);
      setSearching(false);
      return;
    }
    const timer = setTimeout(() => {
      setSearching(true);
      api.user
        .searchUsers(searchQuery)
        .then((res) => {
          if (active && Array.isArray(res)) {
            setSearchResults(res.filter((u) => u.username !== profile.username));
          }
        })
        .catch(() => { if (active) setSearchResults([]); })
        .finally(() => { if (active) setSearching(false); });
    }, 280);
    return () => { active = false; clearTimeout(timer); };
  }, [searchQuery, profile.username]);

  // Active workspace & channel
  const ws = spaces.find((x) => x.id === space);
  const workspaceId = ws?.id;
  const channelId = ws?.channels?.find((c) => c.slug === selected)?.id;
  const activeDm = space === "dm" ? dmContacts.find((c) => c.username === selected) : null;
  const key = space === "dm" ? selected : `${space}:${selected}`;
  const list = useMemo(() => channelMessages[key] || [], [channelMessages, key]);
  const draft = drafts[key] || "";
  const displayed = list.filter((m) =>
    m.text.toLowerCase().includes(search.toLowerCase())
  );

  useEffect(() => {
    let active = true;
    let fetching = false;
    const refresh = async () => {
      if (fetching || document.visibilityState !== "visible") return;
      fetching = true;
      try {
        const data = await api.conversations.list();
        if (!active) return;
        setDmContacts((previous) => data.map((item) => {
          const old = previous.find((contact) => contact.id === item.user_id);
          return { ...old, id: item.user_id, conversationId: item.id, username: item.username, name: item.display_name };
        }));
      } catch { if (active) setNotice("Daftar percakapan tidak dapat diperbarui."); }
      finally { fetching = false; }
    };
    void refresh();
    const timer = window.setInterval(refresh, 15000);
    return () => { active = false; window.clearInterval(timer); };
  }, []);

  useEffect(() => {
    if (space !== "dm" || !activeDm?.conversationId) return;
    const id = activeDm.conversationId;
    let active = true;
    let fetching = false;
    const refresh = async () => {
      if (fetching || document.visibilityState !== "visible") return;
      fetching = true;
      const revision = channelMutationRef.current;
      try {
        const data = await api.conversations.messages(id);
        if (!active || revision !== channelMutationRef.current) return;
        setChannelMessages((previous) => ({ ...previous, [selected]: data.map((message) => ({
          id: message.sequence, authorId: message.author_id, author: message.author_id === accountId ? profile.name : message.author,
          authorUsername: message.username, text: message.text,
          time: new Intl.DateTimeFormat("id-ID", { hour: "2-digit", minute: "2-digit" }).format(new Date(message.created_at)),
        })) }));
        setChannelError("");
      } catch (error) {
        if (active) {
          if (error instanceof ApiError && [401,403,404].includes(error.status)) setChannelMessages((previous) => { const next = { ...previous }; delete next[selected]; return next; });
          setChannelError("Percakapan tidak dapat dimuat. Periksa koneksi atau akses akunmu.");
        }
      } finally { fetching = false; }
    };
    void refresh();
    const timer = window.setInterval(refresh, 10000);
    return () => { active = false; window.clearInterval(timer); };
  }, [space, activeDm?.conversationId, selected, accountId, profile.name]);

  useEffect(() => {
    if (!channelId || !workspaceId) return;
    let active = true;
    let fetching = false;
    const channelKey = `${workspaceId}:${selected}`;
    const fetchMessages = async () => {
      if (fetching) return;
      fetching = true;
      const revision = channelMutationRef.current;
      try {
        const data = await api.channelMessages.list(channelId);
        if (!active || revision !== channelMutationRef.current) return;
        setChannelMessages((previous) => ({ ...previous, [channelKey]: data.map((message: ChannelMessageEntity) => ({
          id: message.sequence,
          authorId: message.author_id,
          author: message.author_id === accountId ? profile.name : message.author,
          authorUsername: message.username,
          text: message.text,
          time: new Intl.DateTimeFormat("id-ID", { hour: "2-digit", minute: "2-digit" }).format(new Date(message.created_at)),
        })) }));
        setChannelError("");
      } catch (error) {
        if (active) {
          if (error instanceof ApiError && [401, 403, 404].includes(error.status)) {
            setChannelMessages((previous) => {
              const next = { ...previous };
              delete next[channelKey];
              return next;
            });
            setChannelError("Channel tidak tersedia atau aksesmu telah berubah.");
          } else setChannelError("Pesan channel tidak dapat dimuat. Memeriksa koneksi kembali…");
        }
      } finally { fetching = false; }
    };
    void fetchMessages();
    const timer = window.setInterval(fetchMessages, 10000);
    return () => { active = false; window.clearInterval(timer); };
  }, [channelId, workspaceId, selected, accountId, profile.name]);

  const markVisibleAsRead = useCallback(() => {
    const body = chatBodyRef.current;
    if (
      !body ||
      !canMarkRead({
        visible: document.visibilityState === "visible",
        focused: document.hasFocus(),
        covered: !!dialog || mobileList,
        filtering: !!search.trim(),
        distanceFromBottom:
          body.scrollHeight - body.scrollTop - body.clientHeight,
      })
    )
      return;
    const latest = [...list].reverse().find((m) => m.authorId ? m.authorId !== accountId : m.author !== profile.name);
    if (!latest) return;
    setReadCursors((prev) =>
      prev[key] === latest.id ? prev : { ...prev, [key]: latest.id }
    );
  }, [key, list, dialog, mobileList, search, profile.name, accountId]);

  useEffect(() => {
    const frame = requestAnimationFrame(markVisibleAsRead);
    const onVisibility = () => markVisibleAsRead();
    document.addEventListener("visibilitychange", onVisibility);
    window.addEventListener("focus", onVisibility);
    window.addEventListener("resize", onVisibility);
    return () => {
      cancelAnimationFrame(frame);
      document.removeEventListener("visibilitychange", onVisibility);
      window.removeEventListener("focus", onVisibility);
      window.removeEventListener("resize", onVisibility);
    };
  }, [markVisibleAsRead]);

  function updateDraft(v: string) {
    setDrafts((x) => ({ ...x, [key]: v }));
  }

  async function send(e?: FormEvent) {
    e?.preventDefault();
    if (!draft.trim() || !selected) return;
    if (ws || activeDm?.conversationId) {
      const destinationId = ws ? channelId : activeDm?.conversationId;
      if (!destinationId || sendingChannelMessage) return;
      setSendingChannelMessage(true);
      setChannelError("");
      const text = draft.trim();
      let submitted = false;
      try {
        const pending = await pendingChannelRequest(sessionStorage, crypto, accountId, destinationId, text);
        submitted = true;
        const m = ws ? await api.channelMessages.send(destinationId, text, pending.key) : await api.conversations.send(destinationId, text, pending.key);
        try { sessionStorage.removeItem(pending.storageKey); } catch {
          setChannelError("Pesan terkirim, tetapi data percobaan kirim belum dapat dibersihkan di browser.");
        }
        channelMutationRef.current += 1;
        setChannelMessages((previous) => ({ ...previous, [key]: [...(previous[key] || []).filter((item) => item.id !== m.sequence), {
          id: m.sequence, authorId: m.author_id, author: profile.name, authorUsername: m.username, text: m.text,
          time: new Intl.DateTimeFormat("id-ID", { hour: "2-digit", minute: "2-digit" }).format(new Date(m.created_at)),
        }].sort((a,b) => a.id-b.id) }));
        setDrafts((previous) => previous[key] === draft ? { ...previous, [key]: "" } : previous);
      } catch (error) {
        if (!submitted) {
          setChannelError(error instanceof Error && error.message.includes("percobaan kirim")
            ? error.message
            : "Pesan belum dikirim karena penyimpanan percobaan kirim tidak tersedia di browser.");
        } else if (error instanceof ApiError && error.status >= 400 && error.status < 500) {
          setChannelError(error.status === 409
            ? "Percobaan kirim tidak lagi dapat diulang. Periksa riwayat pesan."
            : "Pesan ditolak. Periksa akses dan isi pesan sebelum mencoba lagi.");
        } else {
          setChannelError("Status kirim belum dapat dipastikan. Coba kirim lagi tanpa mengubah pesan.");
        }
      } finally { setSendingChannelMessage(false); }
      return;
    }
    setChannelError("Percakapan belum siap. Buka kembali dan coba kirim pesan lagi.");
  }

  function changeMsg(id: number, fn: (m: Msg) => Msg) {
    setMessages((x) => ({
      ...x,
      [key]: (x[key] || []).map((m) => (m.id === id ? fn(m) : m)),
    }));
  }

  function selectSpace(id: string) {
    setSpace(id);
    if (id === "dm") {
      setSelected(dmContacts[0]?.username || "");
    } else {
      const targetWs = spaces.find((w) => w.id === id);
      setSelected(targetWs?.channels?.[0]?.slug || "umum");
    }
    setSearch("");
    setPanel("");
    setMobileList(false);
  }

  async function startDmWithUser(user: {
    id?: string;
    username: string;
    name?: string;
    avatar?: string;
    bio?: string;
    status?: string;
    customStatus?: string;
  }) {
    try {
      let userId = user.id;
      if (!userId) {
        const matches = await api.user.searchUsers(user.username);
        userId = matches.find((item) => item.username.toLowerCase() === user.username.toLowerCase())?.id;
      }
      if (!userId || userId === accountId) throw new Error("Pengguna tidak dapat ditemukan.");
      const conversation = await api.conversations.open(userId);
      const newContact: DMContact = {
        id: userId,
        conversationId: conversation.id,
        name: user.name || user.username,
        username: user.username,
        avatar: user.avatar,
        status: user.status || "online",
        customStatus: user.customStatus,
        bio: user.bio,
      };
      setDmContacts((prev) => [newContact, ...prev.filter((contact) => contact.id !== userId)]);
    setSpace("dm");
    setSelected(user.username);
    setDialog("");
    setViewingUser(null);
    setMobileList(false);
    } catch (error) { setNotice(error instanceof Error ? error.message : "Percakapan tidak dapat dibuka."); }
  }

  // Create Workspace in real PostgreSQL backend
  async function createSpace(e: FormEvent) {
    e.preventDefault();
    if (!newName.trim() || creatingWs) return;
    setCreatingWs(true);
    try {
      setNotice("Menyiapkan workspace...");
      const created = await api.workspaces.create({
        name: newName.trim(),
        description: newDesc.trim(),
      });
      setSpaces((prev) => [...prev, created]);
      setSpace(created.id);
      setSelected(created.channels?.[0]?.slug || "umum");
      setDialog("");
      setNewName("");
      setNewDesc("");
      setNotice("");
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setNotice(err.message);
      } else {
        setNotice("Gagal membuat workspace.");
      }
    } finally {
      setCreatingWs(false);
    }
  }

  // Create Channel in real PostgreSQL backend
  async function createChannel(e: FormEvent) {
    e.preventDefault();
    if (!ws || !newName.trim()) return;
    try {
      const created = await api.workspaces.createChannel(ws.id, {
        name: newName.trim(),
      });
      setSpaces((prev) =>
        prev.map((w) =>
          w.id === ws.id ? { ...w, channels: [...(w.channels || []), created] } : w
        )
      );
      setSelected(created.slug);
      setDialog("");
      setNewName("");
      setNotice("");
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setNotice(err.message);
      } else {
        setNotice("Gagal membuat channel.");
      }
    }
  }

  async function loadWorkspaceInvites(workspaceID: string) {
    try {
      const items = await api.workspaces.listInvites(workspaceID);
      setWorkspaceInvites(items);
    } catch (err) {
      setWorkspaceInvites([]);
      if (err instanceof ApiError && err.status !== 403) setNotice(err.message);
    }
  }

  // Invitation settings stay explicit: a member never receives a fabricated
  // fallback link when the API cannot create a real capability URL.
  async function openWorkspaceInviteModal(targetWs: WorkspaceDetailEntity) {
    setInviteWsTarget(targetWs);
    setDialog("workspace-invite");
    setLoadingInvite(true);
    setNotice("");
    setInviteGeneratedCode("");
    setInviteGeneratedLink("");
    setInviteDurationHours(168);
    setInviteMaxUses(0);
    try {
      await loadWorkspaceInvites(targetWs.id);
    } finally {
      setLoadingInvite(false);
    }
  }

  async function createWorkspaceInvite() {
    if (!inviteWsTarget) return;
    setLoadingInvite(true);
    setNotice("");
    try {
      const res = await api.workspaces.createInvite(inviteWsTarget.id, {
        duration_hours: inviteDurationHours,
        max_uses: inviteMaxUses,
      });
      setInviteGeneratedCode(res.code);
      setInviteGeneratedLink(res.invite_url);
      await loadWorkspaceInvites(inviteWsTarget.id);
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setNotice(err.message);
      } else {
        setNotice("Tautan undangan belum dapat dibuat. Silakan coba lagi.");
      }
    } finally {
      setLoadingInvite(false);
    }
  }

  async function revokeWorkspaceInvite(inviteID: string) {
    if (!inviteWsTarget) return;
    setRevokingInviteID(inviteID);
    setNotice("");
    try {
      await api.workspaces.revokeInvite(inviteWsTarget.id, inviteID);
      setWorkspaceInvites((items) =>
        items.map((item) => item.id === inviteID ? { ...item, status: "revoked", revoked_at: new Date().toISOString() } : item)
      );
    } catch (err) {
      setNotice(err instanceof ApiError ? err.message : "Undangan tidak dapat dicabut.");
    } finally {
      setRevokingInviteID("");
    }
  }

  // Accept workspace invitation from server
  async function acceptWorkspaceJoin() {
    if (!joinWsOffer || joiningWorkspace) return;
    setJoiningWorkspace(true);
    try {
      setNotice("Bergabung ke workspace...");
      const joined = await api.invites.accept(joinWsOffer.code);
      setSpaces((prev) => {
        if (prev.some((w) => w.id === joined.id)) return prev.map(w => w.id === joined.id ? joined : w);
        return [...prev, joined];
      });
      setSpace(joined.id);
      setSelected(joined.channels?.[0]?.slug || "umum");
      setJoinWsOffer(null);
      setNotice("");
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setNotice(err.message);
      } else {
        setNotice("Gagal bergabung ke workspace.");
      }
    } finally { setJoiningWorkspace(false); }
  }

  // Handle manual join via code
  async function handleManualJoin(e: FormEvent) {
    e.preventDefault();
    if (!manualJoinCode.trim()) return;
    const rawInvite = manualJoinCode.trim();
    let cleanCode = rawInvite;
    try {
      const parsed = new URL(rawInvite);
      cleanCode = parsed.searchParams.get("join") || parsed.searchParams.get("join_workspace") || parsed.pathname.match(/^\/invite\/([^/]+)\/?$/)?.[1] || "";
    } catch {
      cleanCode = rawInvite.replace(/^.*[?&](?:join|join_workspace)=([^&]+).*$/, "$1");
    }
    try {
      setNotice("Memeriksa kode undangan...");
      const prev = await api.invites.getPreview(cleanCode);
      setDialog("");
      setManualJoinCode("");
      setJoinWsOffer(prev);
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setNotice(err.message);
      } else {
        setNotice("Kode undangan tidak valid atau kedaluwarsa.");
      }
    }
  }

  // Upload file avatar or banner to real backend storage with instant preview
  async function upload(
    e: React.ChangeEvent<HTMLInputElement>,
    field: "avatar" | "banner"
  ) {
    const file = e.target.files?.[0];
    if (!file) return;

    const localUrl = URL.createObjectURL(file);
    setProfile((x) => ({ ...x, [field]: localUrl }));
    setUploadingField(field);
    setNotice(`Mengunggah ${field === "avatar" ? "foto profil" : "banner"}...`);

    try {
      const media = await api.media.uploadAndFinalize(file, field);
      if (field === "avatar") {
        const updated = await api.user.setAvatar(media.id);
        setProfile((x) => ({
          ...x,
          avatar: updated.avatar_url || media.public_url,
          revision: updated.revision,
        }));
      } else {
        const updated = await api.user.setBanner(media.id);
        setProfile((x) => ({
          ...x,
          banner: updated.banner_url || media.public_url,
          revision: updated.revision,
        }));
      }
      setNotice(
        `${field === "avatar" ? "Foto profil" : "Banner"} berhasil disimpan!`
      );
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setNotice(err.message);
      } else {
        setNotice("Gambar belum dapat disimpan. Silakan coba lagi.");
      }
    } finally {
      setUploadingField(null);
    }
  }

  async function saveProfile() {
    setSavingProfile(true);
    setNotice("Menyimpan perubahan profil...");
    try {
      const updated = await api.user.updateProfile(
        {
          display_name: profile.name,
          username: profile.username,
          bio: profile.bio,
          availability: profile.status,
          custom_status: profile.custom,
        },
        profile.revision
      );
      setProfile((x) => ({
        ...x,
        name: updated.display_name,
        username: updated.username,
        bio: updated.bio || "",
        status: updated.availability || "Tersedia",
        custom: updated.custom_status || "",
        revision: updated.revision,
      }));
      setNotice("Profil berhasil disimpan.");
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setNotice(err.message);
      } else {
        setNotice("Gagal menyimpan profil.");
      }
    } finally {
      setSavingProfile(false);
    }
  }

  async function handleRevokeSession(sessionId: string) {
    try {
      await api.user.revokeSession(sessionId);
      setRealSessions((prev) => prev.filter((s) => s.id !== sessionId));
      setNotice("Sesi berhasil dicabut.");
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setNotice(err.message);
      } else {
        setNotice("Gagal mencabut sesi.");
      }
    }
  }

  async function handleLogout() {
    try {
      await signOutEverywhere();
      router.push("/login");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Gagal keluar. Coba lagi.");
    }
  }

  const dialogs = !!dialog;
  const currentChatName = ws ? `# ${selected}` : activeDm?.name || selected || "Pilih Obrolan";
  const userInviteLink =
    typeof window !== "undefined"
      ? `${window.location.origin}/invite/user/${encodeURIComponent(profile.username)}`
      : `/invite/user/${encodeURIComponent(profile.username)}`;

  return (
    <main className="messenger" data-theme={theme} id="main-content">
      {/* --- SIDE RAIL: NAVIGATION & WORKSPACES --- */}
      <aside
        className={`rail ${railPinned ? "rail-pinned" : ""}`}
        aria-label="Navigasi ruang"
      >
        <div className="rail-heading">
          <Link href="/" className="rail-logo" aria-label="Beranda Relay">
            <div className="app-brand-mark">
              <span />
              <span />
              <span />
              <span />
            </div>
          </Link>
          <span className="rail-heading-copy" aria-hidden="true">
            <b>relay.</b>
            <small>RUANG CERITA</small>
          </span>
          <button
            className="rail-pin"
            type="button"
            onClick={() => setRailPinned(!railPinned)}
            aria-label={railPinned ? "Ringkas sidebar" : "Sematkan sidebar terbuka"}
            aria-pressed={railPinned}
            title={railPinned ? "Ringkas sidebar" : "Sematkan sidebar terbuka"}
          >
            {railPinned ? <ChevronLeft size={16} /> : <ChevronRight size={16} />}
          </button>
        </div>

        <div className="rail-section-label" aria-hidden="true">
          RUANG KAMU
        </div>
        <button
          className={`rail-item ${space === "dm" ? "rail-active" : ""}`}
          onClick={() => selectSpace("dm")}
          aria-label="Pesan pribadi"
          aria-current={space === "dm" ? "page" : undefined}
          title="Pesan pribadi"
        >
          <span className="rail-icon">
            <MessageSquare size={18} strokeWidth={1.8} />
          </span>
          <span className="rail-label" aria-hidden="true">
            Pesan pribadi<small>Percakapan langsung</small>
          </span>
        </button>

        <div className="rail-rule" />
        <div className="rail-section-label" aria-hidden="true">
          WORKSPACE ({spaces.length})
        </div>
        <div className="rail-workspaces">
          {spaces.map((w) => (
            <button
              key={w.id}
              onClick={() => selectSpace(w.id)}
              className={`rail-item ${space === w.id ? "rail-active" : ""}`}
              aria-label={w.name}
              aria-current={space === w.id ? "page" : undefined}
              title={w.name}
            >
              <span className="rail-icon rail-workspace-icon">
                {w.name.slice(0, 2).toUpperCase()}
              </span>
              <span className="rail-label" aria-hidden="true">
                {w.name}
                <small>{w.channels?.length || 1} channel</small>
              </span>
            </button>
          ))}
          <button
            className="rail-item rail-add"
            onClick={() => {
              setNewName("");
              setNewDesc("");
              setDialog("workspace");
            }}
            aria-label="Buat workspace"
          >
            <span className="rail-icon">
              <Plus size={18} />
            </span>
            <span className="rail-label" aria-hidden="true">
              Buat workspace<small>Mulai ruang tim baru</small>
            </span>
          </button>
        </div>

        <div className="rail-spacer" />
        <button
          className="rail-item rail-bottom"
          onClick={() => {
            setPicker("Profil");
            setDialog("settings");
          }}
          aria-label="Pengaturan"
        >
          <span className="rail-icon">
            <Settings size={18} strokeWidth={1.8} />
          </span>
          <span className="rail-label" aria-hidden="true">
            Pengaturan<small>Profil & akun</small>
          </span>
        </button>
        <div className="rail-footer" aria-hidden="true">
          LESS APP. MORE CONVERSATION.
        </div>
      </aside>

      {/* --- CONVERSATIONS SIDEBAR --- */}
      {mobileList && <button className="mobile-scrim" aria-label="Tutup daftar percakapan" onClick={() => setMobileList(false)} />}
      <aside id="conversation-list" className={`conversations ${mobileList ? "mobile-open" : ""}`} aria-label="Daftar percakapan">
        <div className="side-brand">
          <span>
            relay<span className="brand-dot">.</span>
          </span>
          <small>RUANG KAMU</small>
        </div>

        <div className="side-heading">
          <div>
            <span className="side-label">
              {space === "dm" ? "RUANG PERSONAL" : "WORKSPACE"}
            </span>
            <h1>{ws?.name || (space === "dm" ? "Pesan Pribadi" : "Workspace")}</h1>
          </div>
          {ws && (
            <button
              onClick={() => setDialog("workspace-settings")}
              aria-label="Pengaturan workspace"
            >
              <Settings size={18} />
            </button>
          )}
        </div>

        {ws ? (
          <>
            <p className="side-description">{ws.description || "Ruang untuk berbagi cerita dan ide."}</p>
            <button
              className="side-invite"
              onClick={() => openWorkspaceInviteModal(ws)}
            >
              <Users size={14} style={{ display: "inline", verticalAlign: "middle", marginRight: 6 }} />
              Undang anggota <span>INVITE</span>
            </button>

            <div className="side-section">
              <span>CHANNEL</span>
              <button
                onClick={() => {
                  setNewName("");
                  setDialog("channel");
                }}
                aria-label="Buat channel"
              >
                <Plus size={14} />
              </button>
            </div>
            {ws.channels?.map((c) => (
              <button
                key={c.slug}
                className={`chat-row channel-row ${selected === c.slug ? "selected" : ""}`}
                onClick={() => {
                  setSelected(c.slug);
                  setMobileList(false);
                }}
              >
                <Hash size={14} style={{ opacity: 0.6 }} />
                {c.name}
              </button>
            ))}

            <div className="side-section">
              <span>ANGGOTA - {ws.members?.length || 1}</span>
              <button onClick={() => setPanel("members")}>Lihat</button>
            </div>
            {ws.members?.map((m) => (
              <button
                key={m.user_id}
                className="member-row"
                onClick={() => {
                  setViewingUser({
                    name: m.display_name,
                    username: m.username,
                    avatar: m.avatar_url,
                    role: m.role,
                    status: "online",
                  });
                }}
              >
                <LuxuryAvatar name={m.display_name} src={m.avatar_url} size={28} />
                <span>
                  {m.display_name}
                  <small>{m.title || m.role}</small>
                </span>
              </button>
            ))}
          </>
        ) : space !== "dm" && spaces.length === 0 ? (
          <div style={{ padding: "20px 8px", textAlign: "center" }}>
            <p style={{ fontSize: "12px", color: "#8a9686", lineHeight: "1.6" }}>
              Kamu belum memiliki workspace.
            </p>
            <button
              className="primary-button"
              style={{ width: "100%", marginTop: "12px" }}
              onClick={() => {
                setNewName("");
                setNewDesc("");
                setDialog("workspace");
              }}
            >
              <Plus size={14} /> Buat Workspace
            </button>
            <button
              className="secondary-button"
              style={{ width: "100%", marginTop: "8px" }}
              onClick={() => setDialog("join-code")}
            >
              <Key size={14} /> Masukkan Kode
            </button>
          </div>
        ) : (
          <>
            <label className="side-search">
              <Search size={14} />
              <input
                placeholder="Cari kontak obrolan..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </label>

            <div className="side-section">
              <span>PESAN PRIBADI</span>
              <button
                onClick={() => {
                  setSearchQuery("");
                  setSearchResults([]);
                  setDialog("search-users");
                }}
                title="Cari & tambah kontak pengguna asli"
              >
                <UserPlus size={13} style={{ display: "inline", marginRight: 4 }} />
                Cari Teman
              </button>
            </div>

            {dmContacts.length === 0 ? (
              <div style={{ padding: "16px 8px", textAlign: "center" }}>
                <p style={{ fontSize: "11px", color: "#8a9686", lineHeight: "1.6" }}>
                  Belum ada percakapan pribadi.
                </p>
                <button
                  className="secondary-button"
                  style={{ width: "100%", marginTop: "8px" }}
                  onClick={() => {
                    setSearchQuery("");
                    setSearchResults([]);
                    setDialog("search-users");
                  }}
                >
                  <UserPlus size={13} /> Cari teman
                </button>
              </div>
            ) : (
              dmContacts
                .filter(
                  (c) =>
                    c.name.toLowerCase().includes(search.toLowerCase()) ||
                    c.username.toLowerCase().includes(search.toLowerCase())
                )
                .map((contact) => (
                  <button
                    key={contact.username}
                    onClick={() => {
                      setSelected(contact.username);
                      setSearch("");
                      setMobileList(false);
                    }}
                    className={`chat-row ${selected === contact.username ? "selected" : ""}`}
                  >
                    <LuxuryAvatar
                      name={contact.name}
                      src={contact.avatar}
                      size={40}
                      status={contact.status || "online"}
                    />
                    <span className="chat-info">
                      <b>{contact.name}</b>
                      <small>
                        {(channelMessages[contact.username] || []).at(-1)?.text || `@${contact.username}`}
                      </small>
                    </span>
                    {unreadCount(channelMessages[contact.username] || [], readCursors[contact.username], accountId) > 0 && (
                      <i className="unread">
                        {unreadCount(channelMessages[contact.username] || [], readCursors[contact.username], accountId)}
                      </i>
                    )}
                  </button>
                ))
            )}
          </>
        )}

        {/* Current User Bar at Bottom of Sidebar */}
        <button
          className="side-profile"
          onClick={() => {
            setViewingUser({
              name: profile.name,
              username: profile.username,
              bio: profile.bio,
              status: profile.status,
              custom: profile.custom,
              avatar: profile.avatar,
              banner: profile.banner,
              role: "Owner",
            });
          }}
        >
          <LuxuryAvatar
            name={profile.name}
            src={profile.avatar}
            size={40}
            status={profile.status || "online"}
          />
          <span>
            <b>{profile.name}</b>
            <small>@{profile.username}</small>
          </span>
          <span
            className="side-gear"
            onClick={(e) => {
              e.stopPropagation();
              setPicker("Profil");
              setDialog("settings");
            }}
            title="Buka Pengaturan"
          >
            <Settings size={18} />
          </span>
        </button>
      </aside>

      {/* --- MAIN CHAT SECTION --- */}
      <section className="chat-main">
        <header className="chat-header">
          <button
            className="mobile-toggle"
            onClick={() => setMobileList(!mobileList)}
            aria-label={mobileList ? "Tutup daftar percakapan" : "Buka daftar percakapan"}
            aria-expanded={mobileList}
            aria-controls="conversation-list"
          >
            <Menu size={18} />
          </button>
          {ws ? (
            <span className="avatar header-avatar">
              <Hash size={18} />
            </span>
          ) : (
            <LuxuryAvatar
              name={activeDm?.name || selected || "?"}
              src={activeDm?.avatar}
              size={42}
              status={activeDm?.status || "online"}
              onClick={() => {
                if (activeDm) {
                  setViewingUser({
                    name: activeDm.name,
                    username: activeDm.username,
                    avatar: activeDm.avatar,
                    bio: activeDm.bio,
                    status: activeDm.status || "online",
                    custom: activeDm.customStatus,
                    role: "Anggota",
                  });
                }
              }}
            />
          )}
          <div>
            <h2>{currentChatName}</h2>
            <p>{ws ? ws.name : activeDm ? `@${activeDm.username}` : "Pesan pribadi"}</p>
          </div>
          <div className="header-tools">
            <button
              onClick={() => setPanel(panel === "search" ? "" : "search")}
              aria-label="Cari pesan"
            >
              <Search size={16} />
            </button>
            <button
              onClick={() => setPanel(panel === "pins" ? "" : "pins")}
              aria-label="Pesan disematkan"
            >
              <Pin size={16} />
            </button>
            <button
              onClick={() => {
                if (!ws && activeDm) {
                  setViewingUser({
                    name: activeDm.name,
                    username: activeDm.username,
                    avatar: activeDm.avatar,
                    bio: activeDm.bio,
                    status: activeDm.status || "online",
                    custom: activeDm.customStatus,
                  });
                } else {
                  setPanel(panel === "members" ? "" : "members");
                }
              }}
              aria-label={ws ? "Lihat anggota" : "Lihat profil"}
            >
              <Info size={16} />
            </button>
          </div>
        </header>

        {!isConnected && <div className="demo-bar" role="status">Koneksi sedang bermasalah. Sebagian informasi mungkin belum terbaru.</div>}

        {panel === "search" && (
          <div className="search-line">
            <input
              autoFocus
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Cari dalam percakapan..."
            />
            <button
              onClick={() => {
                setPanel("");
                setSearch("");
              }}
            >
              Tutup
            </button>
          </div>
        )}

        {channelError && <div className="demo-bar" role="alert">{channelError}</div>}

        <div className="chat-body" ref={chatBodyRef} onScroll={markVisibleAsRead}>
          {!selected && space === "dm" && dmContacts.length === 0 ? (
            <div className="empty-state-card">
              <div className="empty-state-badge">
                <MessageSquare size={28} strokeWidth={1.5} />
              </div>
              <h3 className="empty-state-title">Mulai Obrolan Pribadi</h3>
              <p className="empty-state-subtitle">
                Kamu belum memiliki teman di daftar pesan. Cari pengguna terdaftar di Relay atau bagikan tautan profil kamu untuk mulai mengobrol!
              </p>
              <div style={{ display: "flex", gap: "10px" }}>
                <button
                  className="primary-button"
                  onClick={() => {
                    setSearchQuery("");
                    setSearchResults([]);
                    setDialog("search-users");
                  }}
                >
                  <UserPlus size={14} /> Cari Pengguna
                </button>
                <button
                  className="secondary-button"
                  onClick={() => {
                    navigator.clipboard.writeText(userInviteLink);
                    setNotice("Tautan profil berhasil disalin!");
                    setTimeout(() => setNotice(""), 3000);
                  }}
                >
                  <Copy size={13} /> Salin Link Profil
                </button>
              </div>
            </div>
          ) : space !== "dm" && spaces.length === 0 ? (
            <div className="empty-workspace-view">
              <div className="empty-state-badge">
                <Users size={28} strokeWidth={1.5} />
              </div>
              <h3 className="empty-state-title">Belum Ada Workspace</h3>
              <p className="empty-state-subtitle">
                Kamu belum membuat atau bergabung dengan workspace apa pun. Buat ruang baru untuk berkolaborasi dengan tim atau bergabung via tautan undangan!
              </p>
              <div style={{ display: "flex", gap: "10px" }}>
                <button
                  className="primary-button"
                  onClick={() => {
                    setNewName("");
                    setNewDesc("");
                    setDialog("workspace");
                  }}
                >
                  <Plus size={14} /> Buat Workspace Pertama
                </button>
                <button
                  className="secondary-button"
                  onClick={() => setDialog("join-code")}
                >
                  <Key size={14} /> Masukkan Kode
                </button>
              </div>
            </div>
          ) : (
            <>
              <div className="conversation-intro">
                <div className="intro-symbol">
                  {ws ? <Hash size={24} /> : <MessageSquare size={24} />}
                </div>
                <h3>
                  {ws
                    ? `Selamat datang di #${selected}`
                    : `Ruang percakapan dengan ${activeDm?.name || selected}.`}
                </h3>
                <p>
                  {ws
                    ? "Mulai berbagi ide dan berkolaborasi di channel ini."
                    : "Ruang untuk percakapan bersama."}
                </p>
              </div>

              <div className="date-divider">
                <span>HARI INI</span>
              </div>

              {displayed.length === 0 && (
                <p className="empty-chat">
                  {search
                    ? "Tidak ada pesan yang cocok."
                    : "Belum ada pesan dalam percakapan ini. Mulailah dengan menyapa!"}
                </p>
              )}

              {displayed.map((m) => (
                <article className="message" key={m.id}>
                  <LuxuryAvatar
                    name={m.author}
                    src={(m.authorId ? m.authorId === accountId : m.author === profile.name) ? profile.avatar : m.authorAvatar}
                    size={38}
                    onClick={() => {
                      setViewingUser({
                        name: m.author,
                        username: m.authorUsername || m.author.toLowerCase(),
                        avatar: (m.authorId ? m.authorId === accountId : m.author === profile.name) ? profile.avatar : m.authorAvatar,
                        status: "online",
                      });
                    }}
                  />
                  <div className="message-content">
                    <div className="message-meta">
                      <b
                        style={{ cursor: "pointer" }}
                        onClick={() => {
                          setViewingUser({
                            name: m.author,
                            username: m.authorUsername || m.author.toLowerCase(),
                            avatar: (m.authorId ? m.authorId === accountId : m.author === profile.name) ? profile.avatar : m.authorAvatar,
                            status: "online",
                          });
                        }}
                      >
                        {m.author}
                      </b>
                      <time>{m.time}</time>
                      {(m.authorId ? m.authorId === accountId : m.author === profile.name) && <span>TERKIRIM</span>}
                    </div>
                    {m.reply && <div className="reply-preview">&#8618; {m.reply}</div>}
                    <p>{m.text}</p>
                    {m.reaction && (
                      <button
                        className="reaction"
                        onClick={() =>
                          changeMsg(m.id, (x) => ({ ...x, reaction: undefined }))
                        }
                      >
                        {m.reaction} 1
                      </button>
                    )}
                    {m.pin && (
                      <small className="pinned-label">&#9831; Disematkan</small>
                    )}
                  </div>
                  {!ws && !activeDm?.conversationId && <div className="message-actions">
                    <button
                      onClick={() =>
                        changeMsg(m.id, (x) => ({
                          ...x,
                          reaction: x.reaction ? undefined : "\u2665",
                        }))
                      }
                      aria-label="Beri reaksi"
                    >
                      &#9825;
                    </button>
                    <button onClick={() => setReply(m)} aria-label="Balas pesan">
                      &#8617;
                    </button>
                    <button
                      onClick={() =>
                        changeMsg(m.id, (x) => ({ ...x, pin: !x.pin }))
                      }
                      aria-label="Sematkan pesan"
                    >
                      <Pin size={13} />
                    </button>
                    {m.author === profile.name && (
                      <>
                        <button
                          onClick={() => {
                            setEditing(m.id);
                            updateDraft(m.text);
                          }}
                          aria-label="Edit pesan"
                        >
                          &#9998;
                        </button>
                        <button
                          onClick={() =>
                            setMessages((x) => ({
                              ...x,
                              [key]: (x[key] || []).filter((y) => y.id !== m.id),
                            }))
                          }
                          aria-label="Hapus pesan"
                        >
                          <X size={14} />
                        </button>
                      </>
                    )}
                  </div>}
                </article>
              ))}
            </>
          )}
        </div>

        {/* --- COMPOSER AREA --- */}
        <div className="composer-area">
          {reply && (
            <div className="composer-context">
              Membalas {reply.author}: {reply.text}
              <button onClick={() => setReply(null)}>
                <X size={14} />
              </button>
            </div>
          )}
          {editing !== null && (
            <div className="composer-context">
              Mengedit pesan{" "}
              <button
                onClick={() => {
                  setEditing(null);
                  updateDraft("");
                }}
              >
                <X size={14} />
              </button>
            </div>
          )}
          {picker && (
            <div className="picker">
              <b>{picker === "emoji" ? "Pilih ekspresi" : "Stiker Relay"}</b>
              <div>
                {(picker === "emoji"
                  ? ["halo!", "semangat", "terima kasih", "keren!", "ide bagus", "siap!"]
                  : ["halo", "senang", "terima kasih", "setuju!"]
                ).map((x) => (
                  <button
                    key={x}
                    onClick={() => {
                      updateDraft(draft + " " + x);
                      setPicker("");
                    }}
                  >
                    {x}
                  </button>
                ))}
              </div>
              <small>Ekspresi interaktif Relay</small>
            </div>
          )}
          <form className="composer" onSubmit={send}>
            <button
              type="button"
              onClick={() => setPicker(picker === "emoji" ? "" : "emoji")}
              aria-label="Pilih emoji"
            >
              <Smile size={18} />
            </button>
            <input
              value={draft}
              onChange={(e) => updateDraft(e.target.value)}
              placeholder={`Tulis pesan ${
                ws ? `di #${selected}` : `untuk ${activeDm?.name || selected}`
              }...`}
              aria-label="Tulis pesan"
              maxLength={4000}
              disabled={!selected || (space === "dm" ? !activeDm?.conversationId : !channelId)}
            />
            <button
              type="button"
              onClick={() => setPicker(picker === "sticker" ? "" : "sticker")}
              aria-label="Pilih stiker"
            >
              <Sparkles size={18} />
            </button>
            <button className="send-button" aria-label="Kirim pesan" disabled={!selected || !draft.trim() || sendingChannelMessage || (space === "dm" ? !activeDm?.conversationId : !channelId)}>
              <SendHorizontal size={17} strokeWidth={2} /><span>Kirim</span>
            </button>
          </form>
          <div className="composer-help">
            ENTER UNTUK MENGIRIM <span>RUANG UNTUK CERITA</span>
          </div>
        </div>
      </section>

      {/* --- DETAIL PANEL (PINS / MEMBERS / PROFILE) --- */}
      {panel && panel !== "search" && (
        <aside className="detail-panel">
          <div className="detail-title">
            <b>
              {panel === "pins"
                ? "Pesan disematkan"
                : ws
                ? "Anggota workspace"
                : "Profil"}
            </b>
            <button onClick={() => setPanel("")} aria-label="Tutup panel">
              <X size={18} />
            </button>
          </div>
          {panel === "pins" ? (
            <>
              {list.filter((m) => m.pin).length ? (
                list
                  .filter((m) => m.pin)
                  .map((m) => (
                    <p key={m.id} className="detail-card">
                      <b>{m.author}</b>
                      <br />
                      {m.text}
                    </p>
                  ))
              ) : (
                <p className="detail-muted">Belum ada pesan yang disematkan.</p>
              )}
            </>
          ) : ws ? (
            <>
              <p className="detail-muted">{ws.members?.length || 1} anggota</p>
              {ws.members?.map((m) => (
                <div
                  className="detail-member"
                  key={m.user_id}
                  style={{ cursor: "pointer" }}
                  onClick={() => {
                    setViewingUser({
                      name: m.display_name,
                      username: m.username,
                      avatar: m.avatar_url,
                      role: m.role,
                      status: "online",
                    });
                  }}
                >
                  <LuxuryAvatar name={m.display_name} src={m.avatar_url} size={30} />
                  <span>
                    <b>{m.display_name}</b>
                    <small>
                      {m.title || "Anggota"} &bull; {m.role}
                    </small>
                  </span>
                </div>
              ))}
              <button
                className="secondary-button"
                onClick={() => setDialog("workspace-settings")}
                style={{ width: "100%", marginTop: "16px" }}
              >
                <Settings size={14} /> Pengaturan workspace
              </button>
            </>
          ) : null}
        </aside>
      )}

      {/* --- MODAL DIALOGS --- */}
      {dialogs && (
        <div
          className="modal-backdrop"
          onMouseDown={(e) => {
            if (e.target === e.currentTarget) setDialog("");
          }}
        >
          <div
            className={`modal ${dialog === "search-users" ? "modal-discovery" : ""}`}
            role="dialog"
            aria-modal="true"
            aria-label="Dialog"
          >
            <button
              className="modal-close"
              onClick={() => {
                setDialog("");
                setNotice("");
              }}
              aria-label="Tutup"
            >
              <X size={20} />
            </button>

            {/* --- CREATE WORKSPACE / CHANNEL --- */}
            {dialog === "workspace" || dialog === "channel" ? (
              <>
                <span className="modal-header-tag">
                  <Sparkles size={12} /> RUANG BARU
                </span>
                <h2>
                  {dialog === "workspace"
                    ? "Buat workspace baru."
                    : "Mulai channel baru."}
                </h2>
                <form
                  onSubmit={dialog === "workspace" ? createSpace : createChannel}
                >
                  <label>
                    Nama {dialog === "workspace" ? "workspace" : "channel"}
                    <input
                      value={newName}
                      onChange={(e) => setNewName(e.target.value)}
                      required
                      maxLength={40}
                      autoFocus
                      placeholder={
                        dialog === "workspace"
                          ? "Misalnya, Studio Antariksa"
                          : "Misalnya, pengumuman"
                      }
                    />
                  </label>
                  {dialog === "workspace" && (
                    <label>
                      Deskripsi
                      <input
                        value={newDesc}
                        onChange={(e) => setNewDesc(e.target.value)}
                        maxLength={100}
                        placeholder="Untuk apa workspace ini digunakan?"
                      />
                    </label>
                  )}
                  {notice && <p role="alert" style={{ color: "#d9534f", fontSize: "11px", marginTop: "8px" }}>{notice}</p>}
                  <button className="primary-button" disabled={creatingWs} style={{ marginTop: "16px" }}>
                    {creatingWs ? (
                      <>
                        <Loader2 size={14} className="animate-spin" /> Sedang membuat...
                      </>
                    ) : (
                      <>
                        <Plus size={14} /> Buat {dialog === "workspace" ? "workspace" : "channel"}
                      </>
                    )}
                  </button>
                </form>
              </>
            ) : dialog === "join-code" ? (
              /* --- MANUAL JOIN CODE MODAL --- */
              <>
                <span className="modal-header-tag">
                  <Key size={12} /> GABUNG WORKSPACE
                </span>
                <h2>Masukkan Kode Undangan.</h2>
                <p>
                  Tempelkan link undangan atau 8-karakter kode undangan yang kamu terima dari teman.
                </p>
                <form onSubmit={handleManualJoin}>
                  <label>
                    Kode atau Link Undangan
                    <input
                      autoFocus
                      value={manualJoinCode}
                      onChange={(e) => setManualJoinCode(e.target.value)}
                      placeholder="Contoh: x7R2bA8Z atau http://.../app?join=x7R2bA8Z"
                      required
                    />
                  </label>
                  {notice && <p role="alert" style={{ color: "#d9534f", fontSize: "11px" }}>{notice}</p>}
                  <button className="primary-button" style={{ marginTop: "16px" }}>
                    <ArrowUpRight size={14} /> Periksa Undangan
                  </button>
                </form>
              </>
            ) : dialog === "workspace-invite" ? (
              <>
                <span className="modal-header-tag">
                  <Users size={12} /> AKSES WORKSPACE
                </span>
                <h2>Undang dengan kendali.</h2>
                <p>
                  Buat tautan baru hanya saat dibutuhkan. Setiap tautan dapat dibatasi, ditinjau, lalu dicabut tanpa mengubah workspace.
                </p>

                {loadingInvite ? (
                  <div style={{ padding: "20px", textAlign: "center", fontSize: "12px", color: "#6a7b6a" }}>
                    Menyelaraskan akses workspace...
                  </div>
                ) : (
                  <>
                    <section className="invite-control-card" aria-label="Buat undangan baru">
                      <div className="invite-control-heading">
                        <span className="invite-control-orb" aria-hidden="true"><UserPlus size={15} /></span>
                        <div><b>Buat tautan baru</b><small>Hak akses dasar sebagai anggota</small></div>
                      </div>
                      <div className="invite-fields">
                        <label>Masa aktif
                          <select value={inviteDurationHours} onChange={(e) => setInviteDurationHours(Number(e.target.value))}>
                            <option value={1}>1 jam</option><option value={24}>24 jam</option><option value={168}>7 hari</option><option value={720}>30 hari</option>
                          </select>
                        </label>
                        <label>Batas bergabung
                          <select value={inviteMaxUses} onChange={(e) => setInviteMaxUses(Number(e.target.value))}>
                            <option value={0}>Tanpa batas</option><option value={1}>1 orang</option><option value={5}>5 orang</option><option value={25}>25 orang</option><option value={100}>100 orang</option>
                          </select>
                        </label>
                      </div>
                      <button className="invite-create-button" type="button" onClick={createWorkspaceInvite}>
                        <Sparkles size={14} /> Buat tautan aman
                      </button>
                    </section>

                    {inviteGeneratedLink && (
                      <section className="invite-result" aria-live="polite">
                        <div><span>TAUTAN BARU SIAP</span><b>{inviteGeneratedCode}</b></div>
                        
                        <div style={{ display: "flex", flexDirection: "column", gap: "10px", marginTop: "12px" }}>
                          <div>
                            <small style={{ color: "#7a8a7a", fontSize: "11px", fontWeight: 600, display: "block", marginBottom: "4px" }}>
                              Halaman undangan:
                            </small>
                            <div className="invite-link-row">
                              <input
                                readOnly
                                className="invite-link-input"
                                value={inviteGeneratedLink}
                                aria-label="Halaman undangan kartu"
                              />
                              <button
                                className="invite-copy-btn"
                                type="button"
                                onClick={() => {
                                  navigator.clipboard?.writeText(inviteGeneratedLink).then(() => {
                                    setCopiedLink(true);
                                    setTimeout(() => setCopiedLink(false), 2500);
                                  });
                                }}
                              >
                                {copiedLink ? <><Check size={12} /> Tersalin</> : <><Copy size={12} /> Salin</>}
                              </button>
                            </div>
                          </div>
                        </div>

                        <small style={{ display: "block", marginTop: "10px", color: "#6a7b6a" }}>
                          Bagikan salah satu tautan di atas. Teman Anda dapat langsung bergabung ke ruang kerja <b>{inviteWsTarget?.name}</b>.
                        </small>
                      </section>
                    )}

                    <section className="invite-ledger" aria-label="Undangan workspace">
                      <div className="invite-ledger-heading"><b>Undangan yang dikelola</b><small>{workspaceInvites.length} tercatat</small></div>
                      {workspaceInvites.length ? workspaceInvites.map((item) => (
                        <div className="invite-ledger-row" key={item.id}>
                          <div><b>{item.status === "active" ? "Aktif" : item.status === "revoked" ? "Dicabut" : item.status === "expired" ? "Kedaluwarsa" : "Penuh"}</b><small>{item.max_uses === 0 ? `${item.uses} bergabung` : `${item.uses}/${item.max_uses} bergabung`} · hingga {new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short" }).format(new Date(item.expires_at))}</small></div>
                          {item.status === "active" ? <button type="button" onClick={() => revokeWorkspaceInvite(item.id)} disabled={revokingInviteID === item.id}>{revokingInviteID === item.id ? "Mencabut…" : "Cabut"}</button> : <span className="invite-ledger-status">{item.status}</span>}
                        </div>
                      )) : <p className="invite-ledger-empty">Belum ada undangan aktif atau riwayat yang dapat dikelola.</p>}
                    </section>
                    {notice && <p role="alert" className="invite-error">{notice}</p>}
                  </>
                )}

                <div style={{ marginTop: "18px" }}>
                  <button
                    className="secondary-button"
                    onClick={() => setDialog("")}
                    style={{ width: "100%" }}
                  >
                    Tutup
                  </button>
                </div>
              </>
            ) : dialog === "search-users" ? (
              /* --- SEARCH REAL USERS MODAL --- */
              <>
                <span className="modal-header-tag">
                  <UserPlus size={12} /> TAMBAH TEMAN REAL
                </span>
                <h2>Mulai dari satu sapaan.</h2>
                <p>
                  Cari teman melalui nama pengguna, atau bagikan tautan profilmu untuk memulai percakapan.
                </p>

                <div className="user-search-box">
                  <div className="user-search-input-wrap">
                    <span className="user-search-icon">
                      <Search size={15} />
                    </span>
                    <input
                      autoFocus
                      placeholder="Cari nama atau nama pengguna"
                      aria-label="Cari nama atau nama pengguna"
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                    />
                  </div>

                  {searching && (
                    <div style={{ padding: "12px", textAlign: "center", fontSize: "11px", color: "#778a78" }}>
                      Mencari pengguna...
                    </div>
                  )}

                  <div className="user-search-results">
                    {searchResults.length > 0 ? (
                      searchResults.map((u) => (
                        <div className="user-result-row" key={u.id}>
                          <div className="user-result-left">
                            <LuxuryAvatar
                              name={u.display_name || u.username}
                              src={u.avatar_url || ""}
                              size={36}
                            />
                            <div className="user-result-names">
                              <b>{u.display_name || u.username}</b>
                              <small>@{u.username}</small>
                            </div>
                          </div>
                          <button
                            className="user-chat-start-btn"
                            onClick={() => {
                              startDmWithUser({
                                id: u.id,
                                username: u.username,
                                name: u.display_name,
                                avatar: u.avatar_url || "",
                                bio: u.bio,
                                status: u.availability || "online",
                                customStatus: u.custom_status || "",
                              });
                            }}
                          >
                            <MessageSquare size={12} style={{ display: "inline", marginRight: 4 }} />
                            Mulai Chat
                          </button>
                        </div>
                      ))
                    ) : searchQuery.trim() && !searching ? (
                      <div style={{ padding: "16px", textAlign: "center", fontSize: "11px", color: "#8a9686" }}>
                        Tidak ada pengguna yang cocok dengan &quot;{searchQuery}&quot;.
                      </div>
                    ) : null}
                  </div>
                </div>

                <div className="invite-box-container">
                  <span style={{ fontSize: "11px", fontWeight: 700, color: "#324836" }}>
                    ATAU BAGIKAN TAUTAN PROFIL KAMU
                  </span>
                  <div className="invite-link-row">
                    <input
                      readOnly
                      className="invite-link-input"
                      value={userInviteLink}
                    />
                    <button
                      className="invite-copy-btn"
                      onClick={() => {
                        navigator.clipboard?.writeText(userInviteLink).then(() => {
                          setCopiedLink(true);
                          setTimeout(() => setCopiedLink(false), 2500);
                        });
                      }}
                    >
                      {copiedLink ? (
                        <>
                          <Check size={12} style={{ display: "inline", marginRight: 4 }} /> Tersalin!
                        </>
                      ) : (
                        <>
                          <Copy size={12} style={{ display: "inline", marginRight: 4 }} /> Salin Tautan
                        </>
                      )}
                    </button>
                  </div>
                  <small style={{ display: "block", marginTop: "8px", color: "#697a6a", fontSize: "10px" }}>
                    Orang yang membuka tautan akan melihat undangan sebelum melanjutkan ke percakapan.
                  </small>
                </div>
              </>
            ) : dialog === "workspace-settings" ? (
              /* --- WORKSPACE SETTINGS --- */
              <>
                <span className="modal-header-tag">
                  <Settings size={12} /> WORKSPACE
                </span>
                <h2>Pengaturan Ruang Tim.</h2>
                {ws && (
                  <>
                    <label>
                      Nama workspace
                      <input
                        value={ws.name}
                        readOnly
                      />
                    </label>
                    <label>
                      Deskripsi
                      <input
                        value={ws.description}
                        readOnly
                      />
                    </label>
                    <div style={{ marginTop: "16px" }}>
                      <span style={{ fontSize: "11px", fontWeight: 700, color: "#48594b" }}>
                        DAFTAR ANGGOTA ({ws.members?.length || 1})
                      </span>
                      {ws.members?.map((m) => (
                        <div className="member-edit" key={m.user_id} style={{ marginTop: "8px" }}>
                          <b>{m.display_name}</b>
                          <span>@{m.username}</span>
                          <span>{m.role}</span>
                        </div>
                      ))}
                    </div>
                  </>
                )}
              </>
            ) : (
              /* --- HIGH-END PROFILE & APP SETTINGS --- */
              <>
                <span className="modal-header-tag">
                  <Settings size={12} /> PENGATURAN AKUN
                </span>
                <h2>Profil & Preferensi.</h2>
                <div className="settings-tabs">
                  {["Profil", "Perangkat", "Preferensi"].map((x) => (
                    <button
                      key={x}
                      onClick={() => setPicker(x)}
                      className={
                        (!picker && x === "Profil") || picker === x
                          ? "active"
                          : ""
                      }
                    >
                      {x}
                    </button>
                  ))}
                </div>

                {picker === "Perangkat" ? (
                  <>
                    <p>Daftar perangkat dan sesi aktif di akun kamu.</p>
                    {realSessions.length > 0 ? (
                      realSessions.map((s) => (
                        <div className="session-row" key={s.id}>
                          <span>
                            {s.device_name}
                            <small>
                              {s.is_current
                                ? "Sesi saat ini"
                                : `IP: ${s.ip_address} • Aktif: ${new Date(
                                    s.last_active_at
                                  ).toLocaleDateString("id-ID")}`}
                            </small>
                          </span>
                          {!s.is_current && (
                            <button onClick={() => handleRevokeSession(s.id)}>
                              Cabut sesi
                            </button>
                          )}
                        </div>
                      ))
                    ) : (
                      <p className="detail-muted">
                        Belum ada sesi tambahan yang terdaftar.
                      </p>
                    )}
                  </>
                ) : picker === "Preferensi" ? (
                  <>
                    <p>Pilih tampilan yang nyaman untuk percakapanmu.</p>
                    <fieldset className="appearance-options">
                      <legend>Tampilan aplikasi</legend>
                      <button type="button" aria-pressed={theme === "light"} onClick={() => changeTheme("light")}>☀ <span>Terang</span></button>
                      <button type="button" aria-pressed={theme === "dark"} onClick={() => changeTheme("dark")}>☾ <span>Gelap</span></button>
                    </fieldset>
                    {notice && <p role="status">{notice}</p>}
                  </>
                ) : (
                  <>
                    {/* Live Interactive Profile Card Editor with Banner & Avatar Overlays */}
                    <div className="settings-preview-wrap">
                      <div
                        className="settings-banner-preview"
                        style={{
                          backgroundImage: profile.banner
                            ? `url(${profile.banner})`
                            : undefined,
                          background: !profile.banner
                            ? getBanner(profile.username)
                            : undefined,
                        }}
                        onClick={() => bannerInputRef.current?.click()}
                      >
                        <div className="settings-upload-overlay">
                          <span className="settings-upload-pill">
                            <Camera size={13} /> Ganti Sampul
                          </span>
                        </div>
                      </div>

                      <div
                        className="settings-avatar-wrap-pos"
                        onClick={() => avatarInputRef.current?.click()}
                      >
                        <LuxuryAvatar
                          name={profile.name}
                          src={profile.avatar}
                          size={72}
                          status={profile.status || "online"}
                        />
                        <div className="settings-avatar-overlay">
                          <Camera size={18} />
                        </div>
                      </div>

                      <div className="settings-preview-info">
                        <div className="settings-preview-name">
                          {profile.name} <small>@{profile.username}</small>
                        </div>
                        <p className="settings-preview-bio">
                          {profile.bio || "Belum ada bio singkat."}
                        </p>
                        {uploadingField && (
                          <div className="uploading-chip">
                            <span className="upload-spinner" />
                            <span>
                              Mengunggah {uploadingField === "avatar" ? "foto" : "banner"}...
                            </span>
                          </div>
                        )}
                      </div>
                    </div>

                    {/* Hidden Inputs for File Upload */}
                    <input
                      ref={avatarInputRef}
                      type="file"
                      style={{ display: "none" }}
                      accept="image/png,image/jpeg,image/webp"
                      onChange={(e) => upload(e, "avatar")}
                    />
                    <input
                      ref={bannerInputRef}
                      type="file"
                      style={{ display: "none" }}
                      accept="image/png,image/jpeg,image/webp"
                      onChange={(e) => upload(e, "banner")}
                    />

                    <div style={{ display: "flex", gap: "10px", marginBottom: "15px" }}>
                      <button
                        type="button"
                        className="secondary-button"
                        style={{ flex: 1, marginTop: 0 }}
                        onClick={() => avatarInputRef.current?.click()}
                      >
                        <Camera size={14} /> Unggah Foto Profil
                      </button>
                      <button
                        type="button"
                        className="secondary-button"
                        style={{ flex: 1, marginTop: 0 }}
                        onClick={() => bannerInputRef.current?.click()}
                      >
                        <ImageIcon size={14} /> Unggah Banner
                      </button>
                    </div>

                    <label>
                      Nama Tampilan
                      <input
                        value={profile.name}
                        onChange={(e) =>
                          setProfile((x) => ({ ...x, name: e.target.value }))
                        }
                      />
                    </label>
                    <label>
                      Username
                      <input
                        value={profile.username}
                        onChange={(e) =>
                          setProfile((x) => ({
                            ...x,
                            username: e.target.value,
                          }))
                        }
                      />
                    </label>
                    <label>
                      Bio Singkat
                      <input
                        value={profile.bio}
                        maxLength={280}
                        placeholder="Tuliskan tentang peran atau keahlianmu"
                        onChange={(e) =>
                          setProfile((x) => ({ ...x, bio: e.target.value }))
                        }
                      />
                    </label>
                    <label>
                      Status Kustom
                      <input
                        value={profile.custom}
                        maxLength={100}
                        placeholder="Contoh: Sedang merilis fitur baru"
                        onChange={(e) =>
                          setProfile((x) => ({ ...x, custom: e.target.value }))
                        }
                      />
                    </label>
                    <label>
                      Ketersediaan
                      <select
                        value={profile.status}
                        onChange={(e) =>
                          setProfile((x) => ({ ...x, status: e.target.value }))
                        }
                      >
                        <option value="online">Tersedia (Online)</option>
                        <option value="busy">Sibuk</option>
                        <option value="away">Jauh (Away)</option>
                      </select>
                    </label>

                    <button
                      className="primary-button"
                      onClick={saveProfile}
                      disabled={savingProfile}
                      style={{ width: "100%", marginTop: "16px" }}
                    >
                      {savingProfile ? (
                        "Menyimpan..."
                      ) : (
                        <>
                          <Check size={14} /> Simpan Profil
                        </>
                      )}
                    </button>
                    {notice && (
                      <p
                        role="status"
                        style={{
                          marginTop: "10px",
                          fontSize: "11px",
                          color: "#304a39",
                          fontWeight: 700,
                        }}
                      >
                        {notice}
                      </p>
                    )}
                  </>
                )}

                <div
                  style={{
                    marginTop: "30px",
                    borderTop: "1px solid #e1e6dc",
                    paddingTop: "15px",
                  }}
                >
                  <button
                    className="secondary-button"
                    style={{ color: "#bc684d", width: "100%" }}
                    onClick={handleLogout}
                  >
                    <LogOut size={14} /> Keluar dari akun (Logout)
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {/* --- PROFILE CARD MODAL --- */}
      {viewingUser && (
        <ProfileCard
          user={viewingUser}
          onClose={() => setViewingUser(null)}
          onMessage={(u) => startDmWithUser(u)}
        />
      )}

      {/* --- WORKSPACE JOIN CARD MODAL (OPENED VIA ?join=[code]) --- */}
      {joinWsOffer && (
        <div className="discord-card-overlay">
          <div className="discord-join-card">
            <div
              className="workspace-glow-icon"
              style={{ background: getPalette(joinWsOffer.workspace_name).bg }}
            >
              {joinWsOffer.workspace_name.slice(0, 2).toUpperCase()}
            </div>
            <span className="join-kicker">AKSES WORKSPACE</span>
            {notice && <p role="status" aria-live="polite" className="invite-error">{notice}</p>}
            <h3 className="join-ws-title">{joinWsOffer.workspace_name}</h3>
            <p className="join-ws-desc">
              {joinWsOffer.workspace_description || "Kamu telah diundang untuk bergabung ke workspace ini. Mulai ngobrol dan berkolaborasi bersama tim!"}
            </p>
            <div className="join-stats-row">
              <span style={{ display: "flex", alignItems: "center", gap: "6px" }}>
                <LuxuryAvatar name={joinWsOffer.inviter_name} src={joinWsOffer.inviter_avatar} size={20} />
                Diundang oleh <b>{joinWsOffer.inviter_name}</b>
              </span>
              <span style={{ display: "flex", alignItems: "center", gap: "4px" }}>
                <Users size={13} style={{ color: "#22c55e" }} />
                {joinWsOffer.member_count} Anggota
              </span>
            </div>
            <div className="join-actions">
              {isConnected ? (
                <button className="join-btn-accept" onClick={acceptWorkspaceJoin} disabled={joiningWorkspace}>
                  <Users size={14} style={{ display: "inline", marginRight: 6 }} />
                  {joiningWorkspace ? "Sedang bergabung…" : "Gabung workspace"}
                </button>
              ) : (
                <button
                  className="join-btn-accept"
                  onClick={() => {
                    if (typeof window !== "undefined") {
                      sessionStorage.setItem("xynoss_redirect", `/app?join=${joinWsOffer.code}`);
                    }
                    router.push(`/login?redirect=${encodeURIComponent(`/app?join=${joinWsOffer.code}`)}`);
                  }}
                >
                  <Users size={14} style={{ display: "inline", marginRight: 6 }} />
                  Masuk untuk Bergabung
                </button>
              )}
              <button
                className="join-btn-decline"
                disabled={joiningWorkspace}
                onClick={() => setJoinWsOffer(null)}
              >
                Tutup Pratinjau
              </button>
            </div>
          </div>
        </div>
      )}
    </main>
  );
}
