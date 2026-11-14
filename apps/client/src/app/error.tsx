"use client";

import { ErrorState } from "./error-state";

export default function ErrorPage({ reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return <ErrorState kind="generic" onRetry={reset} />;
}
