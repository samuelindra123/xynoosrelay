"use client";

import { useEffect, useRef, useState, type PointerEvent } from "react";
import { api, ApiError, type MediaUploadIntent, type ProfileEntity } from "@/lib/api";
import { validateProfileFile, type MediaPurpose } from "./media-validation";

type Upload = { file: File; field: MediaPurpose; preview: string; intent?: MediaUploadIntent; uploaded?: boolean };
type CropCandidate = { file: File; field: MediaPurpose; preview: string; width: number; height: number };
type CropBox = { x: number; y: number; width: number; height: number };
type CropLayout = { x: number; y: number; width: number; height: number };

function mediaValidationMessage(error: ApiError): string | null {
  if (error.status !== 422) return null;
  const detail = error.details?.find((item) => item.field === "file")?.code || "";
  const message = detail.toLowerCase();
  if (message.includes("signature does not match")) return "Isi file tidak cocok dengan tipe gambarnya. Simpan ulang gambar sebagai JPEG, PNG, atau WebP lalu unggah kembali.";
  if (message.includes("decode") || message.includes("corrupt") || message.includes("malformed")) return "File gambar rusak atau tidak dapat dibaca. Simpan ulang gambar lalu coba lagi.";
  if (message.includes("dimension") || message.includes("4096")) return "Dimensi gambar maksimal 4096 × 4096 piksel.";
  if (message.includes("dangerous") || message.includes("svg")) return "Jenis gambar ini tidak aman untuk diunggah. Gunakan JPEG, PNG, atau WebP.";
  if (message.includes("media type") || message.includes("mime")) return "Format isi file tidak didukung. Gunakan JPEG, PNG, atau WebP.";
  return detail ? `File gambar ditolak: ${detail}` : "File gambar ditolak oleh server. Pilih ulang atau simpan ulang sebagai JPEG, PNG, atau WebP.";
}

export default function MediaEditor({ avatar, banner, onSaved, onPendingChange }: {
  avatar?: string; banner?: string; onSaved: (profile: ProfileEntity) => void; onPendingChange: (pending: boolean) => void;
}) {
  const [upload, setUpload] = useState<Upload | null>(null);
  const [cropCandidate, setCropCandidate] = useState<CropCandidate | null>(null);
  const [cropLayout, setCropLayout] = useState<CropLayout | null>(null);
  const [cropBox, setCropBox] = useState<CropBox | null>(null);
  const [busy, setBusy] = useState(false);
  const [progress, setProgress] = useState(0);
  const [stage, setStage] = useState("");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const busyRef = useRef(false);
  const cropStageRef = useRef<HTMLDivElement>(null);
  const cropDragRef = useRef<{ mode: "move" | "resize"; corner: "nw" | "ne" | "sw" | "se"; x: number; y: number; box: CropBox } | null>(null);
  useEffect(() => { onPendingChange(!!upload || busy); }, [upload, busy, onPendingChange]);
  useEffect(() => () => { if (upload?.preview) URL.revokeObjectURL(upload.preview); }, [upload?.preview]);
  useEffect(() => () => { if (cropCandidate?.preview) URL.revokeObjectURL(cropCandidate.preview); }, [cropCandidate?.preview]);
  useEffect(() => {
    if (!cropCandidate) return;
    const stage = cropStageRef.current;
    if (!stage) return;
    const frame = requestAnimationFrame(() => {
      const stageWidth = stage.clientWidth;
      const stageHeight = stage.clientHeight;
      const scale = Math.min(stageWidth / cropCandidate.width, stageHeight / cropCandidate.height);
      const width = cropCandidate.width * scale;
      const height = cropCandidate.height * scale;
      const layout = { x: (stageWidth - width) / 2, y: (stageHeight - height) / 2, width, height };
      const aspect = cropCandidate.field === "avatar" ? 1 : 8 / 3;
      const boxWidth = Math.min(width, height * aspect);
      const boxHeight = boxWidth / aspect;
      setCropLayout(layout);
      setCropBox({ x: layout.x + (width - boxWidth) / 2, y: layout.y + (height - boxHeight) / 2, width: boxWidth, height: boxHeight });
    });
    return () => cancelAnimationFrame(frame);
  }, [cropCandidate]);

  async function select(file: File | undefined, field: MediaPurpose) {
    if (!file || busyRef.current) return;
    if (upload?.intent) { setError("Selesaikan atau buang unggahan tertunda sebelum memilih gambar lain."); return; }
    const invalid = validateProfileFile(file, field);
    if (invalid) { setError(invalid); return; }
    busyRef.current = true; setBusy(true);
    setError(""); setSuccess("");
    try {
      const image = await createImageBitmap(file);
      const valid = image.width > 0 && image.height > 0;
      const width = image.width;
      const height = image.height;
      image.close();
      if (!valid) { setError("Gambar tidak memiliki dimensi yang valid. Pilih file gambar lain."); return; }
      setCropLayout(null); setCropBox(null);
      setCropCandidate({ file, field, preview: URL.createObjectURL(file), width, height });
      setStage("");
    } catch { setError("Gambar tidak dapat dibaca. Pilih file gambar yang valid."); }
    finally { busyRef.current = false; setBusy(false); }
  }

  function updateCropPointer(event: PointerEvent<HTMLDivElement>) {
    const drag = cropDragRef.current;
    if (!drag || !cropLayout || !cropCandidate) return;
    const dx = event.clientX - drag.x;
    const dy = event.clientY - drag.y;
    if (drag.mode === "move") {
      setCropBox({
        ...drag.box,
        x: Math.max(cropLayout.x, Math.min(cropLayout.x + cropLayout.width - drag.box.width, drag.box.x + dx)),
        y: Math.max(cropLayout.y, Math.min(cropLayout.y + cropLayout.height - drag.box.height, drag.box.y + dy)),
      });
      return;
    }
    const aspect = cropCandidate.field === "avatar" ? 1 : 8 / 3;
    const sx = drag.corner.includes("e") ? 1 : -1;
    const sy = drag.corner.includes("s") ? 1 : -1;
    const deltaWidth = (dx * sx * aspect * aspect + dy * sy * aspect) / (1 + aspect * aspect);
    const anchorX = sx > 0 ? drag.box.x : drag.box.x + drag.box.width;
    const anchorY = sy > 0 ? drag.box.y : drag.box.y + drag.box.height;
    const maxWidth = Math.min(
      sx > 0 ? cropLayout.x + cropLayout.width - anchorX : anchorX - cropLayout.x,
      (sy > 0 ? cropLayout.y + cropLayout.height - anchorY : anchorY - cropLayout.y) * aspect,
    );
    const minWidth = Math.min(40, maxWidth);
    const width = Math.max(minWidth, Math.min(maxWidth, drag.box.width + deltaWidth));
    const height = width / aspect;
    setCropBox({ x: sx > 0 ? anchorX : anchorX - width, y: sy > 0 ? anchorY : anchorY - height, width, height });
  }

  async function confirmCrop() {
    if (!cropCandidate || !cropLayout || !cropBox) return;
    const scale = cropLayout.width / cropCandidate.width;
    const outWidth = cropCandidate.field === "avatar" ? 1024 : 2048;
    const outHeight = cropCandidate.field === "avatar" ? 1024 : 768;
    const sourceWidth = cropBox.width / scale;
    const sourceHeight = cropBox.height / scale;
    const sourceX = (cropBox.x - cropLayout.x) / scale;
    const sourceY = (cropBox.y - cropLayout.y) / scale;
    const canvas = document.createElement("canvas");
    canvas.width = outWidth; canvas.height = outHeight;
    const context = canvas.getContext("2d");
    if (!context) { setError("Pratinjau crop tidak dapat diproses. Coba pilih gambar lagi."); return; }
    try {
      const image = await createImageBitmap(cropCandidate.file);
      context.drawImage(image, sourceX, sourceY, sourceWidth, sourceHeight, 0, 0, outWidth, outHeight);
      image.close();
      const blob = await new Promise<Blob>((resolve, reject) => canvas.toBlob(result => result ? resolve(result) : reject(new Error("Crop gambar gagal diproses.")), cropCandidate.file.type, 0.92));
      const outputType = blob.type || cropCandidate.file.type;
      const extension = outputType === "image/png" ? "png" : outputType === "image/webp" ? "webp" : "jpg";
      const file = new File([blob], `${cropCandidate.file.name.replace(/\.[^.]+$/, "")}-crop.${extension}`, { type: outputType, lastModified: Date.now() });
      setUpload({ file, field: cropCandidate.field, preview: URL.createObjectURL(file) });
      setCropCandidate(null); setProgress(0);
      setStage("Crop siap. Simpan untuk mengunggah gambar."); setError("");
    } catch { setError("Crop gambar gagal diproses. Coba pilih gambar lain."); }
  }

  async function save() {
    if (!upload || busyRef.current) return;
    busyRef.current = true; setBusy(true); setError(""); setSuccess("");
    let current = upload;
    try {
      if (current.intent) {
        setStage("Memeriksa media tertunda…");
        const media = await api.media.get(current.intent.media_id);
        if (["rejected", "deleted", "quarantined"].includes(media.status)) throw new Error("Media ditolak atau tidak dapat dipakai. Buang unggahan ini lalu pilih gambar baru.");
        if (media.status === "ready") current = { ...current, uploaded: true };
      } else {
        setStage("Menyiapkan unggahan…");
        const intent = await api.media.requestUpload({ purpose: current.field, content_type: current.file.type, size_bytes: current.file.size });
        current = { ...current, intent }; setUpload(current);
      }
      if (!current.uploaded) {
        setStage("Mengunggah gambar…");
        await api.media.uploadBinary(current.intent!, current.file, setProgress);
        current = { ...current, uploaded: true }; setUpload(current);
      }
      setStage("Memvalidasi dan menyelesaikan media…");
      const media = await api.media.finalize(current.intent!.media_id);
      if (media.status !== "ready") throw new Error("Media masih diproses. Coba selesaikan lagi sebentar.");
      setStage("Memasang gambar ke profil…");
      const attached = current.field === "avatar" ? await api.user.setAvatar(media.id) : await api.user.setBanner(media.id);
      const persisted = await api.user.getProfile();
      const mediaId = current.field === "avatar" ? persisted.avatar_media_id : persisted.banner_media_id;
      if (mediaId !== media.id) throw new Error("Unggahan selesai, tetapi profil belum mengonfirmasi gambar baru. Coba selesaikan lagi.");
      const urlField = current.field === "avatar" ? "avatar_url" : "banner_url";
      const imageUrl = persisted[urlField] || attached[urlField] || media.url;
      if (!imageUrl) throw new Error("Gambar sudah terpasang, tetapi tautan pratinjau belum tersedia. Muat ulang profil untuk melihatnya.");
      onSaved({ ...persisted, [urlField]: imageUrl }); setUpload(null); setStage(""); setSuccess(`${current.field === "avatar" ? "Avatar" : "Banner"} tersimpan dan terpasang di profil.`);
    } catch (err) {
      setError(err instanceof ApiError && err.status === 401 ? "Sesi berakhir. Masuk kembali sebelum mencoba lagi." : err instanceof ApiError ? mediaValidationMessage(err) || err.message : err instanceof Error ? err.message : "Gambar belum dapat disimpan. Coba lagi.");
      setStage("Unggahan belum selesai. Retry melanjutkan media yang sama; finalize yang gagal dapat diulang.");
    } finally { busyRef.current = false; setBusy(false); }
  }

  async function discard() {
    if (!upload || busyRef.current || !window.confirm("Buang pratinjau dan media unggahan yang belum dipasang?")) return;
    busyRef.current = true; setBusy(true); setError("");
    try {
      if (upload.intent) {
        // An attach request may have committed even when its response timed out.
        const profile = await api.user.getProfile();
        if ([profile.avatar_media_id, profile.banner_media_id].includes(upload.intent.media_id)) {
          onSaved(profile); setUpload(null); setStage(""); setSuccess("Gambar sudah tersimpan di profil. Pratinjau disegarkan."); return;
        }
        await api.media.delete(upload.intent.media_id);
      }
      setUpload(null); setStage(""); setSuccess("Unggahan dibuang. Kamu dapat memilih gambar baru.");
    } catch (err) { setError(err instanceof Error ? err.message : "Media belum dapat dibuang. Coba lagi."); }
    finally { busyRef.current = false; setBusy(false); }
  }

  async function remove(field: MediaPurpose) {
    if (busyRef.current || !window.confirm(`Hapus ${field === "avatar" ? "avatar" : "banner"} yang ditampilkan pada pratinjau profil?`)) return;
    busyRef.current = true; setBusy(true); setError(""); setSuccess("");
    try {
      onSaved(field === "avatar" ? await api.user.deleteAvatar() : await api.user.deleteBanner());
      setSuccess(`${field === "avatar" ? "Avatar" : "Banner"} berhasil dihapus.`);
    } catch (err) { setError(err instanceof Error ? err.message : "Gambar gagal dihapus. Coba lagi."); }
    finally { busyRef.current = false; setBusy(false); }
  }

  return <section className="media-editor" aria-label="Media profil" aria-busy={busy}>
    <p>JPEG, PNG, WebP; avatar maksimal 5 MB, banner 10 MB; dimensi maksimal 4096 × 4096.</p>
    <div className="media-controls">
      {(["avatar", "banner"] as const).map(field => <div key={field}>
        <label>{field === "avatar" ? "Pilih avatar" : "Pilih banner"}<input type="file" accept="image/png,image/jpeg,image/webp" disabled={busy || !!upload?.intent} onChange={event => { void select(event.target.files?.[0], field); event.target.value = ""; }} /></label>
        <button type="button" className="secondary-button" disabled={busy || !!upload || !(field === "avatar" ? avatar : banner)} onClick={() => void remove(field)}>Hapus {field}</button>
      </div>)}
    </div>
    {upload && <div className="media-pending">
      {/* Blob preview is local and never treated as the saved profile image. */}
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img src={upload.preview} alt={`Pratinjau ${upload.field} baru`} />
      <p>{upload.file.name}</p>
      <progress max={100} value={progress} aria-label="Progress unggahan" /><span>{progress}%</span>
      <div className="action-row"><button type="button" className="primary-button" disabled={busy} onClick={() => void save()}>{busy ? "Memproses…" : upload.intent ? "Coba selesaikan lagi" : "Simpan gambar"}</button>
        <button type="button" className="secondary-button" disabled={busy} onClick={() => void discard()}>Buang unggahan</button></div>
    </div>}
    {stage && <p role="status">{stage}</p>}
    {error && <p role="alert">{error}</p>}
    {success && <p role="status">{success}</p>}
    {error && upload?.intent && !upload.uploaded && <p>Tautan unggah berlaku 15 menit. Jika kedaluwarsa, buang unggahan lalu pilih file kembali.</p>}
    {cropCandidate && <div className="media-crop-backdrop" onClick={() => setCropCandidate(null)}>
      <section className="media-crop-dialog" role="dialog" aria-modal="true" aria-labelledby="media-crop-title" onClick={event => event.stopPropagation()} onKeyDown={event => { if (event.key === "Escape") setCropCandidate(null); }}>
        <h3 id="media-crop-title">Atur {cropCandidate.field === "avatar" ? "avatar" : "banner"}</h3>
        <p>Geser kotak untuk memilih bagian gambar. Tarik sudut kotak untuk mengubah ukurannya.</p>
        <div
          ref={cropStageRef}
          className="media-crop-stage"
          onKeyDown={event => {
            if (!cropBox || !cropLayout) return;
            const step = event.shiftKey ? 20 : 5;
            if (event.key === "ArrowLeft") { event.preventDefault(); setCropBox(value => value ? { ...value, x: Math.max(cropLayout.x, value.x - step) } : value); }
            if (event.key === "ArrowRight") { event.preventDefault(); setCropBox(value => value ? { ...value, x: Math.min(cropLayout.x + cropLayout.width - value.width, value.x + step) } : value); }
            if (event.key === "ArrowUp") { event.preventDefault(); setCropBox(value => value ? { ...value, y: Math.max(cropLayout.y, value.y - step) } : value); }
            if (event.key === "ArrowDown") { event.preventDefault(); setCropBox(value => value ? { ...value, y: Math.min(cropLayout.y + cropLayout.height - value.height, value.y + step) } : value); }
          }}
          aria-label="Area gambar untuk dipotong"
        >
          {cropLayout && <>
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img className="media-crop-source" src={cropCandidate.preview} alt="" draggable={false} style={{ left: cropLayout.x, top: cropLayout.y, width: cropLayout.width, height: cropLayout.height }} />
            {cropBox && <div
              className="media-crop-selection"
              style={{ left: cropBox.x, top: cropBox.y, width: cropBox.width, height: cropBox.height }}
              onPointerDown={event => { cropDragRef.current = { mode: "move", corner: "se", x: event.clientX, y: event.clientY, box: cropBox }; event.currentTarget.setPointerCapture(event.pointerId); }}
              onPointerMove={updateCropPointer}
              onPointerUp={() => { cropDragRef.current = null; }}
              onPointerCancel={() => { cropDragRef.current = null; }}
              role="button" tabIndex={0}
              aria-label="Area crop terpilih. Geser dengan mouse atau gunakan tombol panah untuk memindahkan."
            >
              {(["nw", "ne", "sw", "se"] as const).map(corner => <span
                key={corner}
                className={`media-crop-handle ${corner}`}
                aria-hidden="true"
                onPointerDown={event => { event.stopPropagation(); cropDragRef.current = { mode: "resize", corner, x: event.clientX, y: event.clientY, box: cropBox }; event.currentTarget.setPointerCapture(event.pointerId); }}
              />)}
            </div>}
          </>}
        </div>
        <div className="media-crop-actions">
          <button type="button" className="secondary-button" onClick={() => setCropCandidate(null)}>Batal</button>
          <button type="button" className="primary-button" disabled={!cropBox} onClick={() => void confirmCrop()}>Gunakan crop</button>
        </div>
      </section>
    </div>}
  </section>;
}
