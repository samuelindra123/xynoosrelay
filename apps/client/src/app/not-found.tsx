import type { Metadata } from "next";
import { ErrorState } from "./error-state";

export const metadata: Metadata = {
  title: "Halaman tidak ditemukan",
  description: "Halaman yang kamu cari tidak tersedia atau sudah dipindahkan.",
};

export default function NotFound() {
  return <ErrorState kind="not_found" />;
}
