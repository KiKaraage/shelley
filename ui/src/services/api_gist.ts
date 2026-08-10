export interface GistStatus {
  gist_id: string | null;
  gist_url: string | null;
}

async function parseResponse(r: Response): Promise<any> {
  const text = await r.text();
  try { return JSON.parse(text); } catch { return { error: text }; }
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
  if (!r.ok) throw new Error(data.message || data.error || `Export failed (${r.status})`);
  return data;
}

// updateGist force-updates an existing gist.
async function updateGist(conversationId: string): Promise<{ gist_id: string; gist_url: string }> {
  const r = await fetch(`/api/conversation/${conversationId}/export-gist`, {
    method: "PATCH",
    headers: { "X-Shelley-Request": "1" },
  });
  const data = await parseResponse(r);
  if (!r.ok) throw new Error(data.message || data.error || `Update failed (${r.status})`);
  return data;
}

export const gistApi = { getGistStatus, exportGist, updateGist };
