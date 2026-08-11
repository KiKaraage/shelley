export interface GistStatus {
  gist_id: string | null;
  gist_url: string | null;
}

// Returns the parsed JSON body, or a { error } wrapper when the body isn't
// valid JSON (e.g. a plain-text 500). Callers read message/error defensively
// since the shape varies by endpoint.
async function parseResponse(r: Response): Promise<Record<string, unknown>> {
  const text = await r.text();
  try {
    const data = JSON.parse(text);
    return typeof data === "object" && data !== null ? data : { error: text };
  } catch {
    return { error: text };
  }
}

class GistError extends Error {
  code: string;
  constructor(code: string, message: string) {
    super(message);
    this.name = "GistError";
    this.code = code;
  }
}

function throwGistError(data: Record<string, unknown>, fallback: string): never {
  const code = typeof data.error === "string" ? data.error : "";
  const msg = typeof data.message === "string" ? data.message : fallback;
  throw new GistError(code, msg);
}

// getGistStatus returns whether a gist exists for this conversation.
async function getGistStatus(conversationId: string): Promise<GistStatus> {
  const r = await fetch(`/api/conversation/${conversationId}/export-gist`);
  if (!r.ok) throw new Error(`Failed to get gist status: ${r.statusText}`);
  return r.json();
}

// exportGist creates (or updates) a gist for this conversation.
async function exportGist(conversationId: string): Promise<{ gist_id: string; gist_url: string }> {
  const r = await fetch(`/api/conversation/${conversationId}/export-gist`, {
    method: "POST",
    headers: { "X-Shelley-Request": "1" },
  });
  const data = await parseResponse(r);
  if (!r.ok) throwGistError(data, `Export failed (${r.status})`);
  return data as unknown as { gist_id: string; gist_url: string };
}

// updateGist force-updates an existing gist.
async function updateGist(conversationId: string): Promise<{ gist_id: string; gist_url: string }> {
  const r = await fetch(`/api/conversation/${conversationId}/export-gist`, {
    method: "PATCH",
    headers: { "X-Shelley-Request": "1" },
  });
  const data = await parseResponse(r);
  if (!r.ok) throwGistError(data, `Update failed (${r.status})`);
  return data as unknown as { gist_id: string; gist_url: string };
}

export const gistApi = { getGistStatus, exportGist, updateGist };
