export type SafeApiErrorKind = "offline" | "not_found" | "rate_limited" | "forbidden" | "request" | "server";

export function safeApiError(status: number): { kind: SafeApiErrorKind; message: string } {
  if (status === 0) return { kind: "offline", message: "Kami belum bisa memuatnya sekarang. Periksa koneksi, lalu coba lagi." };
  if (status === 401) return { kind: "request", message: "Sesi kamu sudah berakhir. Masuk kembali untuk melanjutkan." };
  if (status === 403) return { kind: "forbidden", message: "Kamu belum memiliki akses untuk melakukan ini." };
  if (status === 404) return { kind: "not_found", message: "Konten yang kamu cari tidak tersedia atau sudah dipindahkan." };
  if (status === 409) return { kind: "request", message: "Ada perubahan baru. Muat ulang halaman, lalu coba lagi." };
  if (status === 413) return { kind: "request", message: "Ukuran file terlalu besar untuk diunggah." };
  if (status === 422) return { kind: "request", message: "Sepertinya ada bagian yang perlu diperiksa. Coba lihat kembali isianmu." };
  if (status === 429) return { kind: "rate_limited", message: "Terlalu banyak percobaan. Tunggu sebentar lalu coba lagi." };
  if (status >= 500) return { kind: "server", message: "Ada sedikit kendala di pihak kami. Coba lagi sebentar." };
  return { kind: "request", message: "Belum berhasil kali ini. Periksa kembali, lalu coba lagi." };
}
