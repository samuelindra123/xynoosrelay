import { redirect } from "next/navigation";

// This alias resolves request-specific workspace params and immediately redirects.
export const instant = false;

interface WorkspaceRouteProps {
  params: Promise<{
    id: string;
    slug?: string[];
  }>;
}

export default async function WorkspaceRedirectPage({
  params,
}: WorkspaceRouteProps) {
  const { id } = await params;
  redirect(`/app?workspace=${encodeURIComponent(id)}`);
}
