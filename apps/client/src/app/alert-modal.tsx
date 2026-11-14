"use client";

import "./auth.css";
import { useRef } from "react";
import { useDialogFocus } from "./app/use-dialog-focus";

export type AlertType =
  | "backend_offline"
  | "email_not_setup"
  | "google_not_setup"
  | "rate_limited"
  | "general_error";

export interface AlertModalProps {
  open: boolean;
  type: AlertType;
  title: string;
  description: string;
  solutionHint?: string;
  primaryActionLabel?: string;
  onPrimaryAction?: () => void;
  onClose: () => void;
}

export function AlertModal({
  open,
  type,
  title,
  description,
  solutionHint,
  primaryActionLabel,
  onPrimaryAction,
  onClose,
}: AlertModalProps) {
  const dialogRef = useRef<HTMLDivElement>(null);
  useDialogFocus(dialogRef, open, onClose);
  if (!open) return null;

  const badgeConfig = {
    backend_offline: {
      label: "SEBENTAR ADA KENDALA",
      color: "#d94336",
      bg: "#feebe8",
      border: "#f8b4ad",
    },
    email_not_setup: {
      label: "EMAIL BELUM TERSEDIA",
      color: "#b46914",
      bg: "#fef5e7",
      border: "#fadcb0",
    },
    google_not_setup: {
      label: "GOOGLE BELUM TERSEDIA",
      color: "#235ca7",
      bg: "#edf4fd",
      border: "#bed6fa",
    },
    rate_limited: {
      label: "COBA LAGI SEBENTAR",
      color: "#a35322",
      bg: "#fbf0e8",
      border: "#f3ccb1",
    },
    general_error: {
      label: "PERHATIAN",
      color: "#c0392b",
      bg: "#fdeeed",
      border: "#f9c6c2",
    },
  }[type] || {
    label: "PERHATIAN",
    color: "#4f6354",
    bg: "#eef3ec",
    border: "#d0ded1",
  };

  return (
    <div
      className="alert-modal-backdrop"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
      role="dialog"
      ref={dialogRef}
      tabIndex={-1}
      aria-modal="true"
      aria-labelledby="alert-modal-title"
    >
      <div className="alert-modal-card">
        <button
          className="alert-modal-close"
          onClick={onClose}
          aria-label="Tutup dialog"
        >
          &#10005;
        </button>

        <div className="alert-badge" style={{ backgroundColor: badgeConfig.bg, borderColor: badgeConfig.border }}>
          <span className="alert-badge-dot" style={{ backgroundColor: badgeConfig.color }} />
          <span style={{ color: badgeConfig.color, fontWeight: 800, fontSize: "10px", letterSpacing: "0.08em" }}>
            {badgeConfig.label}
          </span>
        </div>

        <h3 id="alert-modal-title" className="alert-modal-title">
          {title}
        </h3>

        <p className="alert-modal-desc">{description}</p>

        {solutionHint && (
          <div className="alert-modal-hint">
            <strong>Yang bisa kamu coba:</strong> {solutionHint}
          </div>
        )}

        <div className="alert-modal-actions">
          {primaryActionLabel && onPrimaryAction && (
            <button
              type="button"
              className="alert-primary-btn"
              onClick={onPrimaryAction}
            >
              {primaryActionLabel}
            </button>
          )}
          <button type="button" className="alert-secondary-btn" onClick={onClose}>
            Tutup
          </button>
        </div>
      </div>

      <style jsx>{`
        .alert-modal-backdrop {
          position: fixed;
          inset: 0;
          background: rgba(18, 30, 22, 0.72);
          backdrop-filter: blur(10px);
          -webkit-backdrop-filter: blur(10px);
          z-index: 9999;
          display: grid;
          place-items: center;
          padding: 20px;
          animation: alertFadeIn 0.2s cubic-bezier(0.16, 1, 0.3, 1);
        }
        .alert-modal-card {
          position: relative;
          width: min(100%, 480px);
          background: #fcfbf7;
          border: 1px solid #dfe5dc;
          border-radius: 20px;
          padding: 30px;
          box-shadow: 0 25px 60px rgba(15, 27, 19, 0.32);
          animation: alertSlideUp 0.22s cubic-bezier(0.16, 1, 0.3, 1);
        }
        .alert-modal-close {
          position: absolute;
          top: 18px;
          right: 20px;
          background: none;
          border: 0;
          font-size: 16px;
          color: #728373;
          cursor: pointer;
          width: 30px;
          height: 30px;
          border-radius: 50%;
          display: grid;
          place-items: center;
        }
        .alert-modal-close:hover {
          background: #edf2eb;
          color: #26382c;
        }
        .alert-badge {
          display: inline-flex;
          align-items: center;
          gap: 6px;
          padding: 4px 10px;
          border-radius: 20px;
          border: 1px solid;
          margin-bottom: 14px;
        }
        .alert-badge-dot {
          width: 6px;
          height: 6px;
          border-radius: 50%;
        }
        .alert-modal-title {
          font-size: 20px;
          font-weight: 800;
          letter-spacing: -0.04em;
          color: #1e2e22;
          margin: 0 0 10px;
          line-height: 1.3;
        }
        .alert-modal-desc {
          font-size: 13px;
          line-height: 1.65;
          color: #556758;
          margin: 0 0 16px;
        }
        .alert-modal-hint {
          background: #edf3eb;
          border-left: 3px solid #4a6650;
          padding: 10px 14px;
          font-size: 12px;
          color: #293d2c;
          border-radius: 0 8px 8px 0;
          margin-bottom: 14px;
          line-height: 1.6;
        }
        .alert-snippet-box {
          display: flex;
          align-items: center;
          justify-content: space-between;
          background: #24352a;
          color: #e5efe4;
          padding: 10px 14px;
          border-radius: 10px;
          font-family: monospace;
          font-size: 12px;
          margin-bottom: 20px;
          overflow-x: auto;
        }
        .alert-snippet-copy {
          background: #395240;
          color: #fff;
          border: 0;
          font-size: 10px;
          padding: 4px 8px;
          border-radius: 6px;
          cursor: pointer;
          margin-left: 10px;
          flex-shrink: 0;
        }
        .alert-snippet-copy:hover {
          background: #4e6d57;
        }
        .alert-modal-actions {
          display: flex;
          gap: 10px;
          justify-content: flex-end;
          border-top: 1px solid #e7ede5;
          padding-top: 18px;
          margin-top: 10px;
        }
        .alert-primary-btn {
          background: #283e30;
          color: #fff;
          border: 0;
          padding: 10px 18px;
          font-size: 12px;
          font-weight: 700;
          border-radius: 10px;
          cursor: pointer;
        }
        .alert-primary-btn:hover {
          background: #385542;
        }
        .alert-secondary-btn {
          background: #eaeee8;
          color: #3b4e3e;
          border: 0;
          padding: 10px 16px;
          font-size: 12px;
          font-weight: 700;
          border-radius: 10px;
          cursor: pointer;
        }
        .alert-secondary-btn:hover {
          background: #dbe2d9;
        }
        @keyframes alertFadeIn {
          from { opacity: 0; }
          to { opacity: 1; }
        }
        @keyframes alertSlideUp {
          from { transform: translateY(12px) scale(0.97); opacity: 0; }
          to { transform: translateY(0) scale(1); opacity: 1; }
        }
      `}</style>
    </div>
  );
}
