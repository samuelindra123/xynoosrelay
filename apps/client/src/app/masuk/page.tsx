import { redirect } from "next/navigation";

export const instant = false;

export default async function Page({ searchParams }: { searchParams: Promise<{ error?: string; redirect?: string }> }) {
  const { error, redirect: destination } = await searchParams;
  const query = new URLSearchParams();
  if (error) query.set("error", error);
  if (destination) query.set("redirect", destination);
  const suffix = query.toString();
  redirect(`/login${suffix ? `?${suffix}` : ""}`);
}
