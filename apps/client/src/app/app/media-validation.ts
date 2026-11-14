export type MediaPurpose = "avatar" | "banner";

export function validateProfileFile(file: Pick<File, "type" | "size">, purpose: MediaPurpose): string | null {
  if (!["image/jpeg", "image/png", "image/webp"].includes(file.type)) return "Gunakan gambar JPEG, PNG, atau WebP.";
  const maxMB = purpose === "avatar" ? 5 : 10;
  if (file.size <= 0 || file.size > maxMB * 1024 * 1024) return `Ukuran gambar harus lebih dari 0 dan maksimal ${maxMB} MB.`;
  return null;
}
