/* eslint-disable @typescript-eslint/no-explicit-any */
// Tests for api_gist.ts — specifically that error responses don't crash on
// non-JSON bodies (the original 500 bug: "Internal server error" plain text
// fed to r.json()).
import assert from "node:assert/strict";
import { gistApi } from "./api_gist";

// Minimal fetch mock. Each call returns the configured response.
// ok is derived from status (200-299 = ok) — ResponseInit doesn't accept it.
function mockFetch(status: number, body: string) {
  const original = (globalThis as any).fetch;
  (globalThis as any).fetch = async () =>
    new Response(body, {
      status,
      headers: { "Content-Type": status >= 200 && status < 300 ? "application/json" : "text/plain" },
    });
  return () => { (globalThis as any).fetch = original; };
}

// --- exportGist: plain-text error body (the original 500 bug) ---
{
  const restore = mockFetch(500, "Internal server error");
  try {
    await gistApi.exportGist("conv-1");
    assert.fail("should have thrown");
  } catch (e: any) {
    assert.ok(e instanceof Error, "should be an Error");
    assert.ok(e.message.includes("Internal server error"), `message: ${e.message}`);
  } finally {
    restore();
  }
}

// --- exportGist: JSON error with gh_not_auth ---
{
  const restore = mockFetch(502, JSON.stringify({ error: "gh_not_auth", message: "gh not authenticated" }));
  try {
    await gistApi.exportGist("conv-2");
    assert.fail("should have thrown");
  } catch (e: any) {
    assert.ok(e.message.includes("gh not authenticated"), `message: ${e.message}`);
  } finally {
    restore();
  }
}

// --- exportGist: JSON error with gist_needs_name ---
{
  const restore = mockFetch(422, JSON.stringify({ error: "gist_needs_name", message: "Session must have a name" }));
  try {
    await gistApi.exportGist("conv-3");
    assert.fail("should have thrown");
  } catch (e: any) {
    assert.ok(e.message.includes("Session must have a name"), `message: ${e.message}`);
  } finally {
    restore();
  }
}

// --- exportGist: success ---
{
  const restore = mockFetch(200, JSON.stringify({ gist_id: "abc123", gist_url: "https://gisthost.github.io/?abc123" }));
  try {
    const result = await gistApi.exportGist("conv-4");
    assert.equal(result.gist_id, "abc123");
    assert.ok(result.gist_url.includes("abc123"));
  } finally {
    restore();
  }
}

// --- updateGist: plain-text error ---
{
  const restore = mockFetch(500, "Unexpected error");
  try {
    await gistApi.updateGist("conv-5");
    assert.fail("should have thrown");
  } catch (e: any) {
    assert.ok(e.message.includes("Unexpected error"), `message: ${e.message}`);
  } finally {
    restore();
  }
}

// --- getGistStatus: success ---
{
  const restore = mockFetch(200, JSON.stringify({ gist_id: null, gist_url: "" }));
  try {
    const status = await gistApi.getGistStatus("conv-6");
    assert.equal(status.gist_id, null);
    assert.equal(status.gist_url, "");
  } finally {
    restore();
  }
}

console.log("\napi_gist tests passed");
