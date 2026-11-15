import type { APIError } from '@/types/api';
import { supabase } from '@/lib/supabase';

// Same origin in production (the Go server serves the GUI); the dev server
// talks to the API on :8080 unless VITE_UPSTONK_API_URL says otherwise.
export const API_BASE_URL: string =
  import.meta.env.VITE_UPSTONK_API_URL || (import.meta.env.DEV ? 'http://localhost:8080/api/v1' : '/api/v1');

// An invite link's token is kept until the server has admitted the user.
const INVITE_KEY = 'upstonk.inviteToken';

export function rememberInviteToken(token: string) {
  try {
    localStorage.setItem(INVITE_KEY, token);
  } catch {
    /* storage unavailable: the invite only works in this tab */
  }
}

function pendingInviteToken(): string | null {
  try {
    return localStorage.getItem(INVITE_KEY);
  } catch {
    return null;
  }
}

export function forgetInviteToken() {
  try {
    localStorage.removeItem(INVITE_KEY);
  } catch {
    /* ignore */
  }
}

// apiFetch calls the UPSTONK API with the signed-in user's token and throws
// an APIError (with the server's code) on failure.
export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json');
  if (supabase) {
    const { data } = await supabase.auth.getSession();
    if (data.session) headers.set('Authorization', `Bearer ${data.session.access_token}`);
    const invite = pendingInviteToken();
    if (invite) headers.set('X-Invite-Token', invite);
  }

  const response = await fetch(`${API_BASE_URL}${path}`, { ...init, headers });
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw {
      code: errorData.code || 'API_ERROR',
      message: errorData.message || `Request failed with status ${response.status}`,
      requestId: errorData.requestId || `req_${Date.now()}`,
      status: response.status,
    } as APIError & { status: number };
  }
  if (response.status === 204) return undefined as T;
  return response.json();
}
