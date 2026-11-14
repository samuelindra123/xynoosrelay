// Polling replaces the recent window while preserving pages loaded above it.
export function refreshHistory<T extends { id: number }>(previous: T[], recent: T[]): T[] {
  if (!recent.length) return [];
  const oldest = Math.min(...recent.map(item => item.id));
  return [...previous.filter(item => item.id < oldest), ...recent].sort((a, b) => a.id - b.id);
}

export function prependHistory<T extends { id: number }>(previous: T[], older: T[]): T[] {
  const items = new Map(older.map(item => [item.id, item]));
  for (const item of previous) items.set(item.id, item);
  return [...items.values()].sort((a, b) => a.id - b.id);
}
