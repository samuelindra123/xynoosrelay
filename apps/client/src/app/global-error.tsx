"use client";

import "./globals.css";
import { ErrorState } from "./error-state";

export default function GlobalError({ reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <html lang="id">
      <body><ErrorState kind="generic" onRetry={reset} /></body>
    </html>
  );
}
