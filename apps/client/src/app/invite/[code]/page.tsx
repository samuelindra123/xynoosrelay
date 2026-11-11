import type { Metadata } from "next";
import InviteCard from "./InviteCard";

export const instant = false;

interface PageProps {
  params: Promise<{ code: string }>;
}

export async function generateMetadata({
  params,
}: PageProps): Promise<Metadata> {
  const { code } = await params;
  const apiBase =
    process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/v1";

  try {
    const res = await fetch(`${apiBase}/invites/${encodeURIComponent(code)}`, {
      headers: { Accept: "application/json" },
      next: { revalidate: 60 },
    });

    if (res.ok) {
      const json = await res.json();
      const preview = json.data;
      if (preview && preview.workspace_name) {
        const title = `Gabung ke ${preview.workspace_name} | Xynoos Relay`;
        const description =
          preview.workspace_description ||
          `Undangan bergabung ke ruang kerja ${preview.workspace_name} di platform komunikasi Xynoos Relay.`;

        return {
          title,
          description,
          openGraph: {
            title,
            description,
            type: "website",
          },
          twitter: {
            card: "summary",
            title,
            description,
          },
        };
      }
    }
  } catch {
    // Fallback to standard metadata on network failure or server unreachable
  }

  return {
    title: "Undangan Ruang Kerja | Xynoos Relay",
    description: "Undangan bergabung ke ruang kerja di platform komunikasi Xynoos Relay.",
  };
}

export default async function InvitePage({ params }: PageProps) {
  const { code } = await params;
  return <InviteCard code={code} />;
}
