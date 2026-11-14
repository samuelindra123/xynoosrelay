export async function copyText(text: string): Promise<void> {
  if (!navigator.clipboard?.writeText) throw new Error("Penyalinan otomatis tidak tersedia. Pilih dan salin tautannya secara manual.");
  try { await navigator.clipboard.writeText(text); }
  catch { throw new Error("Tautan belum tersalin. Izinkan akses clipboard atau salin tautan secara manual."); }
}
