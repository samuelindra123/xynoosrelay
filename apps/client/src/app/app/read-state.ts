type ReadableMessage = { id: number; author: string; authorId?: string };
export function unreadCount(items: ReadableMessage[], cursor: number | undefined, ownId?: string): number {
  return items.filter(message => (cursor === undefined || message.id > cursor) && (message.authorId && ownId ? message.authorId !== ownId : message.author !== "Kamu")).length;
}
export function canMarkRead(state: { visible: boolean; focused: boolean; covered: boolean; filtering: boolean; distanceFromBottom: number }): boolean {
  return state.visible && state.focused && !state.covered && !state.filtering && state.distanceFromBottom <= 24;
}
