const MARKDOWN_KEY = "shelley-markdown-rendering";
const CONVERSATION_VIEW_KEY = "shelley-conversation-view";

export type MarkdownMode = "off" | "agent" | "all";
export type ConversationViewMode = "all" | "end-of-turn" | "auto-expand";

export function getMarkdownMode(): MarkdownMode {
  const val = localStorage.getItem(MARKDOWN_KEY);
  // Migrate old boolean values
  if (val === "true") return "agent";
  if (val === "false") return "off";
  if (val === "agent" || val === "all" || val === "off") return val;
  return "agent"; // default
}

export function setMarkdownMode(mode: MarkdownMode): void {
  localStorage.setItem(MARKDOWN_KEY, mode);
}

export function getConversationViewMode(): ConversationViewMode {
  const val = localStorage.getItem(CONVERSATION_VIEW_KEY);
  if (val === "end-of-turn") return "end-of-turn";
  if (val === "auto-expand") return "auto-expand";
  return "all";
}

export function setConversationViewMode(mode: ConversationViewMode): void {
  localStorage.setItem(CONVERSATION_VIEW_KEY, mode);
}
