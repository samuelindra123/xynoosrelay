"use client";

import { useCallback, useEffect, useRef, useState, useMemo, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import "./messenger.css";
import { refreshHistory, prependHistory } from "./message-history";
import { unreadCount, canMarkRead } from "./read-state";
import { pendingChannelRequest } from "./channel-retry";
import MediaEditor from "./media-editor";
import SessionSettings from "./session-settings";
import PrivateAccessSettings from "./private-access-settings";
import { useDialogFocus } from "./use-dialog-focus";
import { copyText } from "@/lib/clipboard";
import {
  api,
  type UserSearchResult,
  type WorkspaceDetailEntity,
  type InvitePreviewEntity,
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
  revision?: number;
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

  const normalizedStatus = status?.toLowerCase();
  const statusColor =
    normalizedStatus === "online" || normalizedStatus === "tersedia"
      ? "#22c55e"
      : normalizedStatus === "busy" || normalizedStatus === "sibuk" || normalizedStatus === "dnd"
      ? "#ef4444"
      : normalizedStatus === "idle" || normalizedStatus === "away" || normalizedStatus === "jauh"
      ? "#f59e0b"
      : normalizedStatus === "offline" ? "#94a3b8" : undefined;

  return (
    <div
      className={`luxury-avatar-wrap ${className}`}
      style={{ width: size, height: size }}
      onClick={onClick}
      role={onClick ? "button" : undefined}
      tabIndex={onClick ? 0 : undefined}
      aria-label={onClick ? `Lihat profil ${name}` : undefined}
      onKeyDown={onClick ? event => { if (event.key === "Enter" || event.key === " ") { event.preventDefault(); onClick(); } } : undefined}
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
  const [profileDetails, setProfileDetails] = useState<{ username: string; details: Partial<UserProfile> } | null>(null);
  const [copied, setCopied] = useState(false);
  const [copyError, setCopyError] = useState("");
  const [profileRefreshError, setProfileRefreshError] = useState("");
  const profileRef = useRef<HTMLDivElement>(null);
  const dismiss = useCallback(() => onClose(), [onClose]);
  useDialogFocus(profileRef, true, dismiss);
  useEffect(() => {
    const username = user.username?.replace(/^@/, "").trim();
    if (!username) return;
    let active = true;
    const refresh = () => api.user.searchUsers(username).then((results) => {
      const match = results.find((result) => result.username.toLowerCase() === username.toLowerCase());
      if (active && match) {
        setProfileRefreshError("");
        setProfileDetails({
        username: user.username,
        details: {
          name: match.display_name || user.name,
          username: match.username,
          bio: match.bio ?? user.bio,
          status: match.availability ?? user.status,
          custom: match.custom_status ?? user.custom,
          avatar: match.avatar_url ?? user.avatar,
          banner: match.banner_url ?? user.banner,
        },
        });
      } else if (active) setProfileRefreshError("Profil belum dapat diperbarui. Coba buka kembali nanti.");
    }).catch(() => { if (active) setProfileRefreshError("Profil belum dapat diperbarui. Periksa koneksi lalu coba lagi."); });
    void refresh();
    const timer = window.setInterval(() => void refresh(), 15000);
    return () => { active = false; window.clearInterval(timer); };
  }, [user]);
  const viewedUser = profileDetails?.username === user.username ? { ...user, ...profileDetails.details } : user;
  const bannerBg = viewedUser.banner ? `url(${viewedUser.banner})` : getBanner(viewedUser.username || viewedUser.name);

  function copyId() {
    copyText(`@${viewedUser.username}`).then(() => {
      setCopied(true);
      setCopyError(""); setTimeout(() => setCopied(false), 2000);
    }).catch(err => setCopyError(err.message));
  }

  return (
    <div className="discord-card-overlay" onClick={onClose}>
      <div className="discord-profile-card" ref={profileRef} role="dialog" aria-modal="true" aria-label="Profil pengguna" tabIndex={-1} onClick={(e) => e.stopPropagation()}>
        {copyError && <p role="alert">{copyError}</p>}
        {profileRefreshError && <p role="status">{profileRefreshError}</p>}
        <div
          className="discord-card-banner"
          style={{
            backgroundImage: viewedUser.banner ? `url(${viewedUser.banner})` : bannerBg,
          }}
        >
          <button className="discord-card-close" onClick={onClose} aria-label="Tutup profil">
            <X size={16} />
          </button>
        </div>
        <div className="discord-card-header">
          <div className="discord-avatar-container">
            <LuxuryAvatar
              name={viewedUser.name}
              src={viewedUser.avatar}
              size={76}
              status={viewedUser.status || "online"}
            />
          </div>
          <div className="discord-badges">
            {viewedUser.role === "Owner" ? (
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
            <h3 className="discord-display-name">{viewedUser.name}</h3>
            <span className="discord-username">@{viewedUser.username}</span>
          </div>

          <div className="discord-presence-row" aria-label={`Ketersediaan: ${viewedUser.status || "online"}`}>
            <span className={`discord-presence-dot presence-${(viewedUser.status || "online").toLowerCase()}`} />
            <span>{({ online: "Tersedia", idle: "Sedang pergi", dnd: "Jangan ganggu", offline: "Offline" } as Record<string, string>)[(viewedUser.status || "online").toLowerCase()] || viewedUser.status}</span>
          </div>

          {viewedUser.custom && (
            <div className="discord-custom-status">
              <MessageSquare size={13} style={{ flexShrink: 0, opacity: 0.8 }} />
              <span>{viewedUser.custom}</span>
            </div>
          )}

          <div className="discord-divider" />

          <div>
            <div className="discord-section-title">TENTANG SAYA</div>
            <p className="discord-bio-text">
              {viewedUser.bio || "Pengguna ini belum menambahkan catatan tentang dirinya."}
            </p>
          </div>

          <div className="discord-actions">
            <button
              className="discord-btn-primary"
              onClick={() => onMessage(viewedUser)}
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

  // Profile of current logged-in user
  const [profile, setProfile] = useState<UserProfile>({
    name: "Kamu",
    username: "kamu",
    bio: "",
    status: "online",
    custom: "",
    avatar: "",
    banner: "",
    revision: 1,
  });

  // Persistent state using storage versioning (xynoss_v2_*)
  const [spaces, setSpaces] = useState<WorkspaceDetailEntity[]>([]);
  const [dmContacts, setDmContacts] = useState<DMContact[]>([]);
  const [messageAvatars, setMessageAvatars] = useState<Record<string, string | null>>({});
  const [channelMessages, setChannelMessages] = useState<Record<string, Msg[]>>({});
  const [sendingChannelMessage, setSendingChannelMessage] = useState(false);
  const [channelError, setChannelError] = useState("");
  const [messageAttempt, setMessageAttempt] = useState(0);
  const [messageLoad, setMessageLoad] = useState<Record<string, { loading: boolean; error: string; blocked?: boolean }>>({});
  const [pendingSends, setPendingSends] = useState<Record<string, { text: string; failed: boolean; rejected?: boolean }>>({});
  const sendingRef = useRef(false);
  const [messageBusy, setMessageBusy] = useState<number | null>(null);
  const [hearts, setHearts] = useState<Record<string, boolean | undefined>>({});
  const actionRef = useRef(false);
  const editRevision = useRef(1);
  const [editConflict, setEditConflict] = useState(false);
  const [workspaceLoad, setWorkspaceLoad] = useState<{ id: string; loading: boolean; error: string; status?: number }>({ id: "", loading: false, error: "" });
  const [workspaceAttempt, setWorkspaceAttempt] = useState(0);
  const [spacesLoading, setSpacesLoading] = useState(true);
  const [spacesError, setSpacesError] = useState("");
  const [searchError, setSearchError] = useState("");
  const [searchAttempt, setSearchAttempt] = useState(0);
  const [dmError, setDmError] = useState("");
  const [dmLoading, setDmLoading] = useState(true);
  const [dmAttempt, setDmAttempt] = useState(0);
  const [inviteListError, setInviteListError] = useState("");
  const [readCursors, setReadCursors] = useState<Record<string, number>>({});
  const [loadingHistory, setLoadingHistory] = useState(false);
  const [historyEnds, setHistoryEnds] = useState<Record<string, boolean>>({});
  const readPending = useRef<Set<string>>(new Set());
  const [theme, setTheme] = useState<"light" | "dark">("light");
  const [joiningWorkspace, setJoiningWorkspace] = useState(false);

  // Active navigation state
  const [space, setSpace] = useState("dm"); // "dm" or workspace id
  const [selected, setSelected] = useState(""); // conversation ID in dm, channel slug in workspace
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
  const [newChannelVisibility, setNewChannelVisibility] = useState<"public" | "private">("public");
  const [privateAccessChannel, setPrivateAccessChannel] = useState("");

  const [creatingWs, setCreatingWs] = useState(false);
  const [notice, setNotice] = useState("");
  const [mobileList, setMobileList] = useState(false);
  const [railPinned, setRailPinned] = useState(false);
  const [isConnected, setIsConnected] = useState(false);
  const [savingProfile, setSavingProfile] = useState(false);
  const [profileConflict, setProfileConflict] = useState(false);
  const chatBodyRef = useRef<HTMLDivElement>(null);
  const channelMutationRef = useRef(0);
  const modalRef = useRef<HTMLDivElement>(null);
  const [mediaPending, setMediaPending] = useState(false);
  const closeDialog = useCallback(() => {
    if (mediaPending) { setNotice("Selesaikan atau buang unggahan pada tab Profil sebelum menutup pengaturan."); return; }
    setDialog("");
  }, [mediaPending]);
  const joinRef = useRef<HTMLDivElement>(null);
  const closeJoin = useCallback(() => { if (!joiningWorkspace) setJoinWsOffer(null); }, [joiningWorkspace]);
  useDialogFocus(joinRef, !!joinWsOffer, closeJoin);
  useDialogFocus(modalRef, !!dialog, closeDialog);

  useEffect(() => {
    try {
      const saved = localStorage.getItem("xynoss_app_theme");
      // Hydrate the browser-only preference after the server render.
      if (saved === "dark" || saved === "light") setTheme(saved);
      else if (window.matchMedia("(prefers-color-scheme: dark)").matches) setTheme("dark");
    } catch { /* Appearance remains usable when storage is unavailable. */ }
  }, []);

  function changeTheme(next: "light" | "dark") {
    setTheme(next);
    try { localStorage.setItem("xynoss_app_theme", next); } catch { /* Keep the current selection for this session. */ }
  }

  // 2. Load Real Workspaces from Database Backend (Starts at 0 workspaces for new users)
  const refreshWorkspaces = useCallback(async () => {
    try {
      const list = await api.workspaces.list();
      if (Array.isArray(list)) {
        setSpaces(list);
      }
      setSpacesError("");
    } catch (err) { setSpacesError(err instanceof Error ? err.message : "Workspace belum dapat dimuat."); }
    finally { setSpacesLoading(false); }
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
            status: data.profile.availability || "online",
            custom: data.profile.custom_status || "",
            avatar: data.profile.avatar_url || "",
            banner: data.profile.banner_url || "",
            revision: data.profile.revision || 1,
          });
        }
        refreshWorkspaces();
      } catch {
        if (active) { setIsConnected(false); setNotice("Profil belum dapat dimuat. Buka kembali aplikasi untuk mencoba lagi."); }
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
      setSpace(wsParam);
      window.history.replaceState({}, "", window.location.pathname);
    }

    if (joinCode) {
      api.invites
        .getPreview(joinCode)
        .then((prev) => {
          setJoinWsOffer(prev);
        })
        .catch(err => {
          setNotice(err instanceof ApiError && err.status === 404 ? "Undangan tidak valid, kedaluwarsa, dicabut, atau kuotanya habis. Minta tautan baru." : err instanceof Error ? err.message : "Undangan belum dapat diperiksa. Coba lagi.");
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
                banner: found.banner_url || "",
                status: found.availability || "online",
                customStatus: found.custom_status || "",
                bio: found.bio || "",
              };
              setDmContacts((prev) => {
                return [newContact, ...prev.filter((c) => c.id !== found.id)];
              });
              setSpace("dm");
              setSelected(conversation.id);
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
          setSelected(activeWs.channels[0].slug);
      }
    }
  }, [space, spaces, selected]);

  // 5. Handle Live User Search
  useEffect(() => {
    let active = true;
    if (!searchQuery.trim()) {
      // Clear stale results as the query is cleared.
      setSearchResults([]);
      setSearching(false);
      return;
    }
    const timer = setTimeout(() => {
      setSearching(true);
      setSearchError("");
      api.user
        .searchUsers(searchQuery)
        .then((res) => {
          if (active && Array.isArray(res)) {
            setSearchResults(res.filter((u) => u.username !== profile.username));
          }
        })
        .catch(err => { if (active) { setSearchResults([]); setSearchError(err instanceof Error ? err.message : "Pencarian gagal. Coba lagi."); } })
        .finally(() => { if (active) setSearching(false); });
    }, 280);
    return () => { active = false; clearTimeout(timer); };
  }, [searchQuery, profile.username, searchAttempt]);

  // Active workspace & channel
  const workspaceReady = space !== "dm" && workspaceLoad.id === space && !workspaceLoad.loading && !workspaceLoad.error;
  const ws = workspaceReady ? spaces.find((x) => x.id === space) : undefined;
  const activeChannel = ws?.channels?.find((c) => c.slug === selected);
  const channelId = activeChannel?.id;
  const activeDm = space === "dm" ? dmContacts.find((c) => c.conversationId === selected) : null;
  const key = space === "dm" ? `dm:${activeDm?.conversationId || "none"}` : `${space}:${selected}`;
  const list = useMemo(() => channelMessages[key] || [], [channelMessages, key]);
  const draft = drafts[key] || "";
  const displayed = (space !== "dm" && !workspaceReady ? [] : list).filter((m) =>
    m.text.toLowerCase().includes(search.toLowerCase())
  );
  function messageAvatar(message: Msg) {
    const isSelf = message.authorId ? message.authorId === accountId : message.author === profile.name;
    if (isSelf) return profile.avatar;
    if (message.authorId && Object.prototype.hasOwnProperty.call(messageAvatars, message.authorId)) return messageAvatars[message.authorId] || undefined;
    return message.authorAvatar;
  }
  const messageAuthors = useMemo(() => {
    const authors = new Map<string, string>();
    for (const message of list) {
      if (message.authorId && message.authorId !== accountId && message.authorUsername) {
        authors.set(message.authorId, message.authorUsername);
      }
    }
    return [...authors].map(([id, username]) => ({ id, username }));
  }, [list, accountId]);
  const messageAuthorsKey = messageAuthors.map(author => `${author.id}:${author.username}`).join("|");

  useEffect(() => {
    if (!messageAuthors.length) return;
    let active = true;
    const refreshMessageAvatars = async () => {
      const results = await Promise.allSettled(messageAuthors.map(async author => {
        const users = await api.user.searchUsers(author.username);
        const user = users.find(result => result.id === author.id);
        return user ? { id: author.id, avatar: user.avatar_url || null } : null;
      }));
      if (!active) return;
      const updates = results.flatMap(result => result.status === "fulfilled" && result.value ? [result.value] : []);
      if (updates.length) setMessageAvatars(previous => {
        const next = { ...previous };
        for (const update of updates) next[update.id] = update.avatar;
        return next;
      });
    };
    void refreshMessageAvatars();
    const timer = window.setInterval(() => void refreshMessageAvatars(), 15000);
    return () => { active = false; window.clearInterval(timer); };
  // Refresh avatars when the authors in this conversation change.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [messageAuthorsKey]);

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
        setDmError("");
      } catch (err) { if (active) setDmError(err instanceof Error ? err.message : "Daftar percakapan belum bisa dimuat."); }
      finally { fetching = false; if (active) setDmLoading(false); }
    };
    void refresh();
    const timer = window.setInterval(refresh, 15000);
    return () => { active = false; window.clearInterval(timer); };
  }, [dmAttempt]);

  const dmContactIds = dmContacts.map((contact) => contact.id).join(",");
  useEffect(() => {
    if (!dmContacts.length) return;
    let active = true;
    const refreshPresence = async () => {
      const results = await Promise.allSettled(dmContacts.map(async (contact) => {
        if (!contact.username) return null;
        const users = await api.user.searchUsers(contact.username);
        return users.find((user) => user.id === contact.id) || null;
      }));
      if (!active) return;
      const presenceById = new Map<string, UserSearchResult>();
      results.forEach((result) => {
        if (result.status === "fulfilled" && result.value) presenceById.set(result.value.id, result.value);
      });
      if (presenceById.size) setDmContacts((previous) => previous.map((contact) => {
        const user = presenceById.get(contact.id);
        return user ? {
          ...contact,
          status: user.availability || undefined,
          customStatus: user.custom_status || "",
          bio: user.bio || "",
          avatar: user.avatar_url || contact.avatar,
          banner: user.banner_url || contact.banner,
        } : contact;
      }));
    };
    void refreshPresence();
    const timer = window.setInterval(() => void refreshPresence(), 15000);
    return () => { active = false; window.clearInterval(timer); };
  // Refresh when the set of direct conversation participants changes.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dmContactIds]);

  useEffect(() => {
    if (space === "dm") return;
    let active = true;
    Promise.resolve().then(async () => {
      if (!active) return;
      setWorkspaceLoad({ id: space, loading: true, error: "" });
      try {
        const detail = await api.workspaces.get(space);
        if (active) {
          setSpaces(previous => [...previous.filter(item => item.id !== detail.id), detail]);
          setWorkspaceLoad({ id: space, loading: false, error: "" });
          setSelected(current => detail.channels?.some(channel => channel.slug === current) ? current : detail.channels?.[0]?.slug || "");
        }
      } catch (err) {
        if (!active) return;
        const status = err instanceof ApiError ? err.status : 0;
        if ([401, 403, 404].includes(status)) {
          setSpaces(previous => previous.filter(item => item.id !== space));
          setChannelMessages(previous => Object.fromEntries(Object.entries(previous).filter(([id]) => !id.startsWith(`${space}:`))));
        }
        setWorkspaceLoad({ id: space, loading: false, error: status === 403 ? "Kamu bukan anggota workspace ini. Gunakan undangan untuk bergabung." : status === 404 ? "Workspace tidak ditemukan atau tidak tersedia." : err instanceof Error ? err.message : "Workspace belum dapat dimuat.", status });
      }
    });
    return () => { active = false; };
  }, [space, workspaceAttempt]);

  useEffect(() => {
    const destination = channelId || activeDm?.conversationId;
    if (!destination) return;
    let active = true;
    let fetching = false;
    const refresh = async () => {
      if (fetching || document.visibilityState !== "visible" || actionRef.current) return;
      fetching = true;
      const revision = channelMutationRef.current;
      try {
        const data = channelId ? await api.channelMessages.list(destination) : await api.conversations.messages(destination);
        if (!active || revision !== channelMutationRef.current || actionRef.current) return;
        setChannelMessages(previous => ({ ...previous, [key]: refreshHistory(previous[key] || [], data.map(message => ({
          id: message.sequence, revision: message.revision || 1, authorId: message.author_id,
          author: message.author_id === accountId ? profile.name : message.author,
          authorUsername: message.username, text: message.text,
          time: new Intl.DateTimeFormat("id-ID", { hour: "2-digit", minute: "2-digit" }).format(new Date(message.created_at)),
        }))) }));
        setMessageLoad(previous => ({ ...previous, [key]: { loading: false, error: "" } }));
        setHistoryEnds(previous => previous[key] === undefined ? { ...previous, [key]: data.length < 100 } : previous);
      } catch (err) {
        if (!active) return;
        const blocked = err instanceof ApiError && [401, 403, 404].includes(err.status);
        if (blocked) setChannelMessages(previous => ({ ...previous, [key]: [] }));
        setMessageLoad(previous => ({ ...previous, [key]: { loading: false, blocked, error: blocked ? "Percakapan tidak tersedia atau kamu tidak lagi memiliki akses." : err instanceof Error ? err.message : "Pesan belum dapat dimuat." } }));
      } finally { fetching = false; }
    };
    void refresh();
    const timer = window.setInterval(refresh, 10000);
    return () => { active = false; window.clearInterval(timer); };
  }, [channelId, activeDm?.conversationId, key, accountId, profile.name, messageAttempt]);

  const readDestination = channelId || activeDm?.conversationId;
  const isWorkspace = !!ws;
  useEffect(() => {
    if (!readDestination) return;
    let active = true;
    const client = isWorkspace ? api.channelMessages : api.conversations;
    void client.read(readDestination).then(state => {
      if (active) setReadCursors(previous => ({ ...previous, [key]: Math.max(previous[key] || 0, state.last_read_sequence) }));
    }).catch(() => { if (active) setChannelError("Tanda pesan terbaca belum bisa diperbarui. Coba lagi sebentar."); });
    return () => { active = false; };
  }, [readDestination, isWorkspace, key, messageAttempt]);

  async function loadOlder() {
    if (!readDestination || loadingHistory || !list.length) return;
    const targetKey = key;
    const before = list[0].id;
    const body = chatBodyRef.current;
    const height = body?.scrollHeight || 0;
    const top = body?.scrollTop || 0;
    setLoadingHistory(true);
    try {
      const revision = channelMutationRef.current;
      const data = isWorkspace ? await api.channelMessages.list(readDestination, before) : await api.conversations.messages(readDestination, before);
      if (revision !== channelMutationRef.current) { setChannelError("Riwayat berubah saat dimuat. Coba muat pesan sebelumnya lagi."); return; }
      const older: Msg[] = data.map(message => ({ id: message.sequence, revision: message.revision || 1, authorId: message.author_id, author: message.author_id === accountId ? profile.name : message.author, authorUsername: message.username, text: message.text, time: new Intl.DateTimeFormat("id-ID", {hour:"2-digit",minute:"2-digit"}).format(new Date(message.created_at)) }));
      setChannelMessages(previous => ({ ...previous, [targetKey]: prependHistory(previous[targetKey] || [], older) }));
      setHistoryEnds(previous => ({ ...previous, [targetKey]: data.length < 100 }));
      requestAnimationFrame(() => { if (body && body.dataset.conversation === targetKey) body.scrollTop = top + body.scrollHeight - height; });
    } catch (err) {
      if (err instanceof ApiError && [401,403,404].includes(err.status)) { setChannelMessages(previous => ({ ...previous, [targetKey]: [] })); setMessageLoad(previous => ({ ...previous, [targetKey]: { loading: false, blocked: true, error: "Percakapan tidak tersedia atau akses kamu telah berubah." } })); }
      else setChannelError("Pesan yang lebih lama belum bisa dimuat. Coba lagi.");
    }
    finally { setLoadingHistory(false); }
  }

  const markVisibleAsRead = useCallback(() => {
    const body = chatBodyRef.current;
    if (
      !body ||
      !canMarkRead({
        visible: document.visibilityState === "visible",
        focused: document.hasFocus(),
        covered: !!dialog || !!viewingUser || !!joinWsOffer || mobileList,
        filtering: !!search.trim(),
        distanceFromBottom:
          body.scrollHeight - body.scrollTop - body.clientHeight,
      })
    )
      return;
    const latest = [...list].reverse().find((m) => m.authorId ? m.authorId !== accountId : m.author !== profile.name);
    if (!latest) return;
    if (!readDestination || (readCursors[key] || 0) >= latest.id || readPending.current.has(key)) return;
    readPending.current.add(key);
    const client = isWorkspace ? api.channelMessages : api.conversations;
    void client.read(readDestination, latest.id).then(state => {
      setReadCursors(previous => ({ ...previous, [key]: Math.max(previous[key] || 0, state.last_read_sequence) }));
    }).catch(() => setChannelError("Tanda pesan terbaca belum tersimpan. Kami akan mencoba lagi nanti."))
      .finally(() => readPending.current.delete(key));
  }, [key, list, dialog, viewingUser, joinWsOffer, mobileList, search, profile.name, accountId, readDestination, readCursors, isWorkspace]);

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

  async function send(e?: FormEvent, retry = false) {
    e?.preventDefault();
    if ((retry ? !pendingSends[key]?.failed : !draft.trim() || !!pendingSends[key]) || !selected || editConflict || messageLoad[key]?.blocked || actionRef.current) return;
    if (ws || activeDm?.conversationId) {
      const destinationId = ws ? channelId : activeDm?.conversationId;
      if (!destinationId || sendingRef.current) return;
      sendingRef.current = true;
      setSendingChannelMessage(true);
      setChannelError("");
      const text = retry ? pendingSends[key].text : draft.trim();
      let submitted = false;
      try {
        if (editing !== null) {
          const revision = editRevision.current;
          channelMutationRef.current += 1;
          try {
            const saved = ws ? await api.channelMessages.edit(destinationId, editing, text, revision) : await api.conversations.edit(destinationId, editing, text, revision);
            setChannelMessages(previous => ({ ...previous, [key]: (previous[key] || []).map(item => item.id === editing ? { ...item, text: saved.text, revision: saved.revision } : item) }));
            setEditing(null);
            setDrafts(previous => ({ ...previous, [key]: "" }));
          } catch (err) {
            if (err instanceof ApiError && [409, 412].includes(err.status)) { setEditConflict(true); setMessageAttempt(value => value + 1); setChannelError("Pesan sudah berubah. Draf edit dipertahankan. Muat versi terbaru sebelum mengedit kembali."); }
            else setChannelError(err instanceof Error ? err.message : "Pesan belum bisa diperbarui. Coba lagi.");
          }
          finally { channelMutationRef.current += 1; setSendingChannelMessage(false); sendingRef.current = false; }
          return;
        }
        setPendingSends(previous => ({ ...previous, [key]: { text, failed: false } }));
        if (!retry) setDrafts(previous => previous[key] === draft ? { ...previous, [key]: "" } : previous);
        const pending = await pendingChannelRequest(sessionStorage, crypto, accountId, destinationId, text);
        submitted = true;
        const m = ws ? await api.channelMessages.send(destinationId, text, pending.key) : await api.conversations.send(destinationId, text, pending.key);
        try { sessionStorage.removeItem(pending.storageKey); } catch {
          setChannelError("Pesanmu sudah terkirim. Muat ulang percakapan untuk menyegarkan tampilannya.");
        }
        channelMutationRef.current += 1;
        setChannelMessages((previous) => ({ ...previous, [key]: [...(previous[key] || []).filter((item) => item.id !== m.sequence), {
          id: m.sequence, revision: m.revision || 1, authorId: m.author_id, author: profile.name, authorUsername: m.username, text: m.text,
          time: new Intl.DateTimeFormat("id-ID", { hour: "2-digit", minute: "2-digit" }).format(new Date(m.created_at)),
        }].sort((a,b) => a.id-b.id) }));
        setPendingSends(previous => { const next = { ...previous }; delete next[key]; return next; });
      } catch (error) {
        setPendingSends(previous => ({ ...previous, [key]: { text, failed: true, rejected: error instanceof ApiError && [400, 401, 403, 404, 413, 422].includes(error.status) } }));
        if (!submitted) {
          setChannelError(error instanceof Error ? error.message : "Pesan belum terkirim. Coba lagi.");
        } else if (error instanceof ApiError && error.status >= 400 && error.status < 500) {
          setChannelError(error.status === 409
            ? "Percobaan kirim berbenturan atau terlalu lama untuk diulang. Periksa riwayat; jangan membuat kiriman baru untuk pesan yang sama."
            : error.message);
        } else {
          setChannelError("Belum bisa dipastikan apakah pesan terkirim. Periksa percakapan sebelum mencoba lagi.");
        }
      } finally { setSendingChannelMessage(false); sendingRef.current = false; }
      return;
    }
    setChannelError("Percakapan belum bisa dibuka. Buka kembali, lalu coba kirim pesan lagi.");
  }

  async function mutateMessage(message: Msg, action: "delete" | "heart-add" | "heart-remove") {
    if (!readDestination || actionRef.current || messageLoad[key]?.blocked) return;
    if (action === "delete" && !window.confirm("Hapus pesan ini?")) return;
    actionRef.current = true; setMessageBusy(message.id); channelMutationRef.current += 1; setChannelError("");
    const heartKey = `${key}:${message.id}`;
    const previousHeart = hearts[heartKey];
    if (action === "delete") setChannelMessages(previous => ({ ...previous, [key]: (previous[key] || []).filter(item => item.id !== message.id) }));
    else setHearts(previous => ({ ...previous, [heartKey]: action === "heart-add" }));
    try {
      const client = isWorkspace ? api.channelMessages : api.conversations;
      if (action === "delete") await client.delete(readDestination, message.id);
      else await client.react(readDestination, message.id, action === "heart-add");
      if (action === "delete" && editing === message.id) { setEditing(null); setEditConflict(false); }
      setNotice(action === "delete" ? "Pesan berhasil dihapus." : action === "heart-add" ? "Heart berhasil ditambahkan." : "Heart kamu berhasil dihapus.");
    } catch (err) {
      const blocked = err instanceof ApiError && [401,403,404].includes(err.status);
      if (blocked) { setChannelMessages(previous => ({ ...previous, [key]: [] })); setMessageLoad(previous => ({ ...previous, [key]: { loading: false, blocked: true, error: "Percakapan tidak tersedia atau akses kamu telah berubah." } })); }
      else if (action === "delete") setChannelMessages(previous => ({ ...previous, [key]: prependHistory(previous[key] || [], [message]) }));
      if (action !== "delete") setHearts(previous => ({ ...previous, [heartKey]: previousHeart }));
      setChannelError(err instanceof Error ? err.message : "Perubahan pesan belum berhasil. Coba lagi.");
      setMessageAttempt(value => value + 1);
    } finally { channelMutationRef.current += 1; actionRef.current = false; setMessageBusy(null); }
  }

  function selectSpace(id: string) {
    setSpace(id);
    setEditing(null); setEditConflict(false); setChannelError("");
    if (id === "dm") {
      setSelected(dmContacts[0]?.conversationId || "");
    } else {
      const targetWs = spaces.find((w) => w.id === id);
      setSelected(targetWs?.channels?.[0]?.slug || "");
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
    banner?: string;
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
        banner: user.banner,
        status: user.status || "online",
        customStatus: user.customStatus,
        bio: user.bio,
      };
      setDmContacts((prev) => [newContact, ...prev.filter((contact) => contact.id !== userId)]);
    setSpace("dm");
    setSelected(conversation.id);
    setDialog("");
    setViewingUser(null);
    setMobileList(false);
    } catch (error) { setNotice(error instanceof ApiError ? error.message : "Belum bisa membuka percakapan sekarang. Coba lagi sebentar."); }
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
      setSelected(created.channels?.[0]?.slug || "");
      setDialog("");
      setNewName("");
      setNewDesc("");
      setNotice("");
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setNotice(err.message);
      } else {
        setNotice("Belum bisa membuat ruang baru. Coba lagi sebentar.");
      }
    } finally {
      setCreatingWs(false);
    }
  }

  // Create Channel in real PostgreSQL backend
  async function createChannel(e: FormEvent) {
    e.preventDefault();
    if (!ws || !newName.trim() || creatingWs) return;
    setCreatingWs(true);
    try {
      const created = await api.workspaces.createChannel(ws.id, {
        name: newName.trim(),
        visibility: newChannelVisibility,
      });
      setSpaces((prev) =>
        prev.map((w) =>
          w.id === ws.id ? { ...w, channels: [...(w.channels || []), created] } : w
        )
      );
      setSelected(created.slug);
      setDialog("");
      setNewName("");
      setNewChannelVisibility("public");
      setNotice("");
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setNotice(err.message);
      } else {
        setNotice("Belum bisa membuat channel baru. Coba lagi sebentar.");
      }
    } finally { setCreatingWs(false); }
  }

  function openPrivateAccess(channelId: string) {
    if (ws?.owner_user_id !== accountId) return;
    setPrivateAccessChannel(channelId); setDialog("private-access"); setNotice("");
  }

  async function loadWorkspaceInvites(workspaceID: string) {
    setInviteListError("");
    try {
      const items = await api.workspaces.listInvites(workspaceID);
      setWorkspaceInvites(items);
    } catch (err) {
      setWorkspaceInvites([]);
      setInviteListError(err instanceof ApiError && err.status === 403 ? "Kamu tidak memiliki izin mengelola daftar undangan." : err instanceof Error ? err.message : "Daftar undangan gagal dimuat.");
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
    if (!inviteWsTarget || loadingInvite) return;
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
        setNotice("Tautan undangan belum berhasil dibuat. Coba lagi sebentar.");
      }
    } finally {
      setLoadingInvite(false);
    }
  }

  async function revokeWorkspaceInvite(inviteID: string) {
    if (!inviteWsTarget || revokingInviteID || !window.confirm("Cabut undangan ini? Tautan tidak dapat dipakai lagi.")) return;
    setRevokingInviteID(inviteID);
    setNotice("");
    try {
      await api.workspaces.revokeInvite(inviteWsTarget.id, inviteID);
      setWorkspaceInvites((items) =>
        items.map((item) => item.id === inviteID ? { ...item, status: "revoked", revoked_at: new Date().toISOString() } : item)
      );
      setNotice("Undangan berhasil dicabut.");
    } catch (err) {
      setNotice(err instanceof ApiError ? err.message : "Belum bisa membatalkan undangan. Coba lagi sebentar.");
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
      setSelected(joined.channels?.[0]?.slug || "");
      setJoinWsOffer(null);
      setNotice("");
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setNotice(err.message);
      } else {
        setNotice("Belum bisa bergabung ke ruang ini. Coba lagi sebentar.");
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
        setNotice("Kode undangan belum bisa diperiksa. Pastikan kodenya benar, lalu coba lagi.");
      }
    }
  }

  async function saveProfile() {
    if (savingProfile || profileConflict || mediaPending) return;
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
        username: updated.username || x.username,
        bio: updated.bio || "",
        status: updated.availability || "online",
        custom: updated.custom_status || "",
        revision: updated.revision,
      }));
      setProfileConflict(false);
      setNotice("Profil berhasil disimpan.");
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        if (err.status === 412) { setProfileConflict(true); setNotice("Profil berubah di perangkat lain. Draf tetap tersedia. Muat profil terbaru sebelum menyimpan lagi."); }
        else setNotice(err.message);
      } else {
        setNotice("Profil belum berhasil disimpan. Coba lagi sebentar.");
      }
    } finally {
      setSavingProfile(false);
    }
  }

  async function handleLogout() {
    try {
      await signOutEverywhere();
      router.push("/login");
    } catch {
      setNotice("Logout belum terkonfirmasi oleh server. Periksa koneksi lalu coba keluar lagi.");
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
              onClick={() => { setDialog("workspace-settings"); setWorkspaceAttempt(value => value + 1); }}
              aria-label="Pengaturan workspace"
            >
              <Settings size={18} />
            </button>
          )}
        </div>

        {spacesLoading && <p role="status">Memuat workspace...</p>}
        {spacesError && <div role="alert"><p>{spacesError}</p><button className="secondary-button" onClick={() => { setSpacesLoading(true); void refreshWorkspaces(); }}>Coba muat workspace lagi</button></div>}
        {space !== "dm" && !workspaceReady && <div className="workspace-state" role={workspaceLoad.error ? "alert" : "status"}>
          <p>{workspaceLoad.id === space && workspaceLoad.error ? workspaceLoad.error : "Memuat detail workspace..."}</p>
          {workspaceLoad.error && <><button className="secondary-button" onClick={() => setWorkspaceAttempt(value => value + 1)}>Coba lagi</button><button className="secondary-button" onClick={() => setDialog("join-code")}>Gunakan undangan</button></>}
        </div>}
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
                  setSelected(c.slug); setEditing(null); setEditConflict(false); setChannelError("");
                  setMobileList(false);
                }}
              >
                {c.visibility === "private" ? <Shield size={13} style={{ opacity: 0.7 }} /> : <Hash size={14} style={{ opacity: 0.6 }} />}
                {c.name}
              </button>
            ))}
            {ws.owner_user_id === accountId && activeChannel?.visibility === "private" && (
              <button className="side-invite" onClick={() => void openPrivateAccess(activeChannel.id)}>
                <Users size={14} style={{ display:"inline", marginRight:6 }} /> Kelola akses channel privat
              </button>
            )}

            <div className="side-section">
              <span>ANGGOTA - {ws.members?.length || 0}</span>
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
        ) : space !== "dm" ? (
          <div style={{ padding: "20px 8px", textAlign: "center" }}>
            <p style={{ fontSize: "12px", color: "#8a9686", lineHeight: "1.6" }}>
              {workspaceLoad.error ? "Workspace ini belum dapat dibuka." : "Menyiapkan workspace..."}
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

            {dmLoading && <p role="status">Memuat percakapan...</p>}
            {dmError && <div role="alert"><p>{dmError}</p><button className="secondary-button" onClick={() => { setDmError(""); setDmLoading(true); setDmAttempt(value => value + 1); }}>Coba muat percakapan lagi</button></div>}
            {!dmLoading && !dmError && dmContacts.length === 0 ? (
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
                    key={contact.conversationId || contact.id}
                    onClick={() => {
                      setSelected(contact.conversationId || ""); setEditing(null); setEditConflict(false); setChannelError("");
                      setSearch("");
                      setMobileList(false);
                    }}
                    className={`chat-row ${selected === contact.conversationId ? "selected" : ""}`}
                  >
                    <LuxuryAvatar
                      name={contact.name}
                      src={contact.avatar}
                      size={40}
                      status={contact.status}
                    />
                    <span className="chat-info">
                      <b>{contact.name}</b>
                      <small>
                        {(channelMessages[`dm:${contact.conversationId}`] || []).at(-1)?.text || (contact.username ? `@${contact.username}` : "Percakapan pribadi")}
                      </small>
                    </span>
                    {unreadCount(channelMessages[`dm:${contact.conversationId}`] || [], readCursors[`dm:${contact.conversationId}`], accountId) > 0 && (
                      <i className="unread">
                        {unreadCount(channelMessages[`dm:${contact.conversationId}`] || [], readCursors[`dm:${contact.conversationId}`], accountId)}
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
              status={activeDm?.status}
              onClick={() => {
                if (activeDm) {
                  setViewingUser({
                    name: activeDm.name,
                    username: activeDm.username,
                    avatar: activeDm.avatar,
                    banner: activeDm.banner,
                    bio: activeDm.bio,
                    status: activeDm.status,
                    custom: activeDm.customStatus,
                    role: "Anggota",
                  });
                }
              }}
            />
          )}
          <div>
            <h2>{currentChatName}</h2>
            <p>{ws ? ws.name : activeDm?.username ? `@${activeDm.username}` : "Pesan pribadi"}</p>
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
                    banner: activeDm.banner,
                    bio: activeDm.bio,
                    status: activeDm.status,
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

        {!isConnected && <div className="demo-bar" role="alert">Profil belum dapat dimuat.<button onClick={() => window.location.reload()}>Coba lagi</button></div>}

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

        {notice && !dialog && <div className="demo-bar" role="status">{notice}<button onClick={() => setNotice("")} aria-label="Tutup pemberitahuan"><X size={14} /></button></div>}
        {channelError && <div className="demo-bar" role="alert">{channelError}<button onClick={() => { setMessageAttempt(value => value + 1); setChannelError(""); }}>Muat ulang percakapan</button></div>}
        {readDestination && (!messageLoad[key] || messageLoad[key].loading) && <p className="demo-bar" role="status">Memuat pesan...</p>}
        {messageLoad[key]?.error && <div className="demo-bar" role="alert">{messageLoad[key].error}<button onClick={() => setMessageAttempt(value => value + 1)}>Coba lagi</button></div>}

        {space !== "dm" && !workspaceReady && <div className="workspace-state" role={workspaceLoad.error ? "alert" : "status"}><p>{workspaceLoad.error || "Memuat workspace..."}</p>{workspaceLoad.error && <><button className="secondary-button" onClick={() => setWorkspaceAttempt(value => value + 1)}>Coba lagi</button><button className="secondary-button" onClick={() => setDialog("join-code")}>Gunakan undangan</button></>}</div>}
        {ws && !ws.channels?.length && <p className="workspace-state">Belum ada channel yang dapat kamu akses. Hubungi owner untuk akses channel privat.</p>}
        <div className="chat-body" ref={chatBodyRef} data-conversation={key} onScroll={markVisibleAsRead}>
          {readDestination && list.length > 0 && !historyEnds[key] && <button className="secondary-button" onClick={() => void loadOlder()} disabled={loadingHistory}>{loadingHistory ? "Memuat riwayat..." : "Muat pesan sebelumnya"}</button>}
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
                    void copyText(userInviteLink).then(() => setNotice("Tautan profil berhasil disalin!")).catch(err => setNotice(err.message));
                  }}
                >
                  <Copy size={13} /> Salin Link Profil
                </button>
              </div>
            </div>
          ) : space !== "dm" && (!workspaceReady || !selected) ? (
            <div className="empty-workspace-view">
              <div className="empty-state-badge">
                <Users size={28} strokeWidth={1.5} />
              </div>
              <h3 className="empty-state-title">{workspaceReady ? "Belum ada channel yang dapat dibuka" : "Workspace belum tersedia"}</h3>
              <p className="empty-state-subtitle">
                Gunakan undangan untuk bergabung. Jika sudah menjadi anggota, hubungi owner untuk mendapatkan akses channel.
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
                  <Plus size={14} /> Buat Workspace
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
                <span>RIWAYAT PERCAKAPAN</span>
              </div>

              {displayed.length === 0 && messageLoad[key] && !messageLoad[key].error && (
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
                    src={messageAvatar(m)}
                    size={38}
                    onClick={() => {
                      setViewingUser({
                        name: m.author,
                        username: m.authorUsername || m.author.toLowerCase(),
                        avatar: messageAvatar(m),
                        status: "online",
                      });
                    }}
                  />
                  <div className="message-content">
                    <div className="message-meta">
                      <b
                        role="button" tabIndex={0}
                        onKeyDown={event => { if (event.key === "Enter" || event.key === " ") { event.preventDefault(); event.currentTarget.click(); } }}
                        style={{ cursor: "pointer" }}
                        onClick={() => {
                          setViewingUser({
                            name: m.author,
                            username: m.authorUsername || m.author.toLowerCase(),
                            avatar: messageAvatar(m),
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
                    {hearts[`${key}:${m.id}`] && <small className="heart-feedback" role="status">? Heart kamu ditambahkan pada sesi ini</small>}
                    {m.pin && (
                      <small className="pinned-label">&#9831; Disematkan</small>
                    )}
                  </div>
                  <div className="message-actions">
                    <button
                      disabled={messageBusy !== null}
                      onClick={() => void mutateMessage(m, "heart-add")}
                      aria-label="Tambahkan heart"
                    >?</button>
                    <button disabled={messageBusy !== null} onClick={() => void mutateMessage(m, "heart-remove")} aria-label="Hapus heart kamu">??</button>
                    <button onClick={() => setReply(m)} aria-label="Balas pesan">
                      &#8617;
                    </button>
                    <button
                      onClick={() => setNotice("Fitur sematkan pesan segera hadir.")}
                      aria-label="Sematkan pesan"
                    >
                      <Pin size={13} />
                    </button>
                    {m.authorId === accountId && (
                      <>
                        <button
                          onClick={() => {
                            if (pendingSends[key]) return;
                            setEditing(m.id); editRevision.current = m.revision || 1; setEditConflict(false);
                            updateDraft(m.text);
                          }}
                          disabled={sendingChannelMessage || !!pendingSends[key]}
                          aria-label="Edit pesan"
                        >
                          &#9998;
                        </button>
                        <button
                          disabled={messageBusy !== null}
                          onClick={() => void mutateMessage(m, "delete")}
                          aria-label="Hapus pesan"
                        >
                          <X size={14} />
                        </button>
                      </>
                    )}
                  </div>
                </article>
              ))}
            </>
          )}
        </div>

        {/* --- COMPOSER AREA --- */}
        <div className="composer-area">
          {pendingSends[key] && <div className="pending-message" role="status"><p>{pendingSends[key].text}</p><small>{pendingSends[key].rejected ? "Server menolak pengiriman. Periksa kembali draf atau akses akun." : pendingSends[key].failed ? "Status pengiriman belum dipastikan. Coba lagi sebelum mengirim draf baru." : "Mengirim... Kamu tetap dapat menulis draf berikutnya."}</small>{pendingSends[key].failed && <button className="secondary-button" disabled={sendingChannelMessage || !!messageLoad[key]?.blocked} onClick={() => void send(undefined, true)}>Coba kirim lagi</button>}
            {pendingSends[key].rejected && <button className="secondary-button" onClick={() => { setDrafts(previous => ({ ...previous, [key]: previous[key]?.trim() ? previous[key] : pendingSends[key].text })); setPendingSends(previous => { const next = { ...previous }; delete next[key]; return next; }); }}>{draft.trim() ? "Singkirkan pesan yang ditolak" : "Perbaiki draf"}</button>}
          </div>}
          {editConflict && <button className="secondary-button" disabled={sendingChannelMessage} onClick={async () => {
            if (!readDestination || editing === null || !window.confirm("Ganti draf edit dengan versi pesan terbaru?")) return;
            try {
              const page = isWorkspace ? await api.channelMessages.list(readDestination, editing + 1) : await api.conversations.messages(readDestination, editing + 1);
              const latest = page.find(message => message.sequence === editing);
              if (latest) {
                editRevision.current = latest.revision || 1; updateDraft(latest.text); setEditConflict(false); setChannelError("");
                setChannelMessages(previous => ({ ...previous, [key]: (previous[key] || []).map(message => message.id === editing ? { ...message, text: latest.text, revision: latest.revision } : message) }));
              } else { setChannelError("Pesan sudah dihapus. Draf edit dipertahankan sampai kamu membatalkannya."); }
            } catch (err) { setChannelError(err instanceof Error ? err.message : "Versi terbaru belum dapat dimuat."); }
          }}>Gunakan versi terbaru</button>}
          {readDestination && readCursors[key] !== undefined && <small className="read-receipt">Posisi baca kamu tersimpan hingga pesan #{readCursors[key]}. Status baca penerima belum tersedia.</small>}
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
                  setEditing(null); setEditConflict(false);
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
          <form className="composer" onSubmit={event => void send(event)}>
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
            <button className="send-button" aria-label="Kirim pesan" disabled={!!pendingSends[key] || !!messageLoad[key]?.blocked || editConflict || !selected || !draft.trim() || sendingChannelMessage || (space === "dm" ? !activeDm?.conversationId : !channelId)}>
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
              <p className="detail-muted">{ws.members?.length || 0} anggota</p>
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
                onClick={() => { setDialog("workspace-settings"); setWorkspaceAttempt(value => value + 1); }}
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
            if (e.target === e.currentTarget) closeDialog();
          }}
        >
          <div
            className={`modal ${dialog === "search-users" ? "modal-discovery" : ""}`}
            ref={modalRef}
            tabIndex={-1}
            role="dialog"
            aria-modal="true"
            aria-label="Dialog"
          >
            <button
              className="modal-close"
              onClick={() => {
                closeDialog();
                if (!mediaPending) setNotice("");
              }}
              aria-label="Tutup"
            >
              <X size={20} />
            </button>

            {/* --- CREATE WORKSPACE / CHANNEL --- */}
            {dialog === "private-access" && ws?.owner_user_id === accountId && <PrivateAccessSettings workspace={ws} channelId={privateAccessChannel} />}
            {dialog === "private-access" ? null : dialog === "workspace" || dialog === "channel" ? (
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
                  {dialog === "channel" && (
                    <label>
                      Akses channel
                      <select value={newChannelVisibility} onChange={event => setNewChannelVisibility(event.target.value as "public" | "private")}>
                        <option value="public">Publik untuk anggota workspace</option>
                        <option value="private">Privat untuk pembuat dan owner</option>
                      </select>
                    </label>
                  )}
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
                  Tempelkan tautan atau kode undangan yang kamu terima dari teman.
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
                      <button className="invite-create-button" disabled={loadingInvite} type="button" onClick={createWorkspaceInvite}>
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
                                  copyText(inviteGeneratedLink).then(() => {
                                    setCopiedLink(true);
                                    setTimeout(() => setCopiedLink(false), 2500);
                                  }).catch(err => setNotice(err.message));
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
                      {inviteListError && <div role="alert"><p>{inviteListError}</p><button className="secondary-button" onClick={() => inviteWsTarget && void loadWorkspaceInvites(inviteWsTarget.id)}>Coba muat daftar lagi</button></div>}
                      {!inviteListError && (workspaceInvites.length ? workspaceInvites.map((item) => (
                        <div className="invite-ledger-row" key={item.id}>
                          <div><b>{item.status === "active" ? "Aktif" : item.status === "revoked" ? "Dicabut" : item.status === "expired" ? "Kedaluwarsa" : "Penuh"}</b><small>{item.max_uses === 0 ? `${item.uses} bergabung` : `${item.uses}/${item.max_uses} bergabung`} · hingga {new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short" }).format(new Date(item.expires_at))}</small></div>
                          {item.status === "active" ? <button type="button" onClick={() => revokeWorkspaceInvite(item.id)} disabled={revokingInviteID === item.id}>{revokingInviteID === item.id ? "Mencabut…" : "Cabut"}</button> : <span className="invite-ledger-status">{item.status}</span>}
                        </div>
                      )) : <p className="invite-ledger-empty">Belum ada undangan aktif atau riwayat yang dapat dikelola.</p>)}
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

                  {searchError && <div role="alert"><p>{searchError}</p><button className="secondary-button" onClick={() => setSearchAttempt(value => value + 1)}>Coba cari lagi</button></div>}
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
                                banner: u.banner_url || "",
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
                    ) : searchQuery.trim() && !searching && !searchError ? (
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
                        copyText(userInviteLink).then(() => {
                          setCopiedLink(true);
                          setTimeout(() => setCopiedLink(false), 2500);
                        }).catch(err => setNotice(err.message));
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
                {!workspaceReady && <div role={workspaceLoad.error ? "alert" : "status"}><p>{workspaceLoad.error || "Memuat detail workspace..."}</p>{workspaceLoad.error && <button className="secondary-button" onClick={() => setWorkspaceAttempt(value => value + 1)}>Coba lagi</button>}</div>}
                {ws && (
                  <>
                    <p role="status">Akses aktif ? {ws.owner_user_id === accountId ? "Owner" : ws.members?.find(member => member.user_id === accountId)?.role || "Anggota"}</p>
                    <p>Owner: <b>{ws.members?.find(member => member.user_id === ws.owner_user_id)?.display_name || ws.owner_user_id}</b></p>
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
                        DAFTAR ANGGOTA ({ws.members?.length || 0})
                      </span>
                      {!ws.members?.length && <p>Belum ada anggota yang ditampilkan.</p>}
                      {ws.members?.map((m) => (
                        <div className="member-edit" key={m.user_id} style={{ marginTop: "8px" }}>
                          <b>{m.display_name}</b>
                          <span>@{m.username}</span>
                          <span>{m.role}</span>
                        </div>
                      ))}
                    </div>
                    <h3>Channel yang dapat kamu akses</h3>
                    {!ws.channels?.length && <p>Belum ada channel yang dapat diakses.</p>}
                    <div className="workspace-channel-list">{ws.channels?.map(channel => <div className="session-row" key={channel.id}>
                      <button className="secondary-button" onClick={() => { setSelected(channel.slug); setEditing(null); setEditConflict(false); setDialog(""); }}># {channel.name}</button>
                      <span>{channel.visibility === "private" ? "Privat ? memiliki akses" : "Publik ? anggota workspace"}</span>
                      {ws.owner_user_id === accountId && channel.visibility === "private" && <button onClick={() => openPrivateAccess(channel.id)}>Kelola akses</button>}
                    </div>)}</div>
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
                {profileConflict && <button className="secondary-button" onClick={async () => {
                  if (!window.confirm("Ganti draf profil dengan data terbaru dari server?")) return;
                  try { const me = await api.user.getMe(); setProfile({ name: me.profile.display_name, username: me.profile.username || me.user.username || "", bio: me.profile.bio, status: me.profile.availability, custom: me.profile.custom_status || "", avatar: me.profile.avatar_url || "", banner: me.profile.banner_url || "", revision: me.profile.revision }); setProfileConflict(false); setNotice("Profil terbaru dimuat."); }
                  catch (err) { setNotice(err instanceof Error ? err.message : "Profil belum dapat dimuat."); }
                }}>Muat profil terbaru</button>}
                <div className="settings-tabs">
                  {["Profil", "Perangkat", "Preferensi"].map((x) => (
                    <button
                      key={x}
                      disabled={mediaPending}
                      aria-pressed={picker === x || (!picker && x === "Profil")}
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
                  <SessionSettings />
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
                            : getBanner(profile.username),
                        }}

                      >
                        <div className="settings-upload-overlay">
                          <span className="settings-upload-pill">
                            Pratinjau banner
                          </span>
                        </div>
                      </div>

                      <div
                        className="settings-avatar-wrap-pos"

                      >
                        <LuxuryAvatar
                          name={profile.name}
                          src={profile.avatar}
                          size={72}
                          status={profile.status || "online"}
                        />
                        <div className="settings-avatar-overlay">
                          Avatar
                        </div>
                      </div>

                      <div className="settings-preview-info">
                        <div className="settings-preview-name">
                          {profile.name} <small>@{profile.username}</small>
                        </div>
                        <p className="settings-preview-bio">
                          {profile.bio || "Belum ada bio singkat."}
                        </p>
                      </div>
                    </div>

                    <MediaEditor avatar={profile.avatar} banner={profile.banner} onPendingChange={setMediaPending} onSaved={updated => setProfile(previous => ({
                      ...previous, avatar: updated.avatar_media_id ? updated.avatar_url || previous.avatar : "", banner: updated.banner_media_id ? updated.banner_url || previous.banner : "", revision: updated.revision,
                    }))} />

                    <label>
                      Nama Tampilan
                      <input
                      value={profile.name}
                        maxLength={64}
                        disabled={savingProfile}
                        onChange={(e) =>
                          setProfile((x) => ({ ...x, name: e.target.value }))
                        }
                      />
                    </label>
                    <label>
                      Username
                      <input
                        value={profile.username}
                        minLength={3}
                        maxLength={32}
                        pattern="[a-zA-Z0-9_]{3,32}"
                        disabled={savingProfile}
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
                        <option value="dnd">Sibuk</option>
                        <option value="idle">Jauh (Away)</option>
                        <option value="offline">Offline</option>
                      </select>
                    </label>

                    <button
                      className="primary-button"
                      onClick={saveProfile}
                      disabled={savingProfile || profileConflict || mediaPending || !isConnected}
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
                    disabled={mediaPending}
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
          <div className="discord-join-card" ref={joinRef} role="dialog" aria-modal="true" aria-label="Terima undangan workspace" tabIndex={-1}>
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
