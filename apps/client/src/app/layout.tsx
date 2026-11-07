import type { Metadata } from "next";
import { Header, Footer } from "./shared";
import "./globals.css";
export const metadata: Metadata = { title: { default: "XYNOSS Relay | Messaging without the weight.", template: "%s | XYNOSS Relay" }, description: "Ruang yang lebih ringan untuk percakapan yang berarti. Kenali XYNOSS Relay." };
export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) { return <html lang="id"><body><a className="skip-link" href="#main-content">Langsung ke konten</a><Header /><div id="main-content">{children}</div><Footer /></body></html>; }
