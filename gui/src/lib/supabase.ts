import { createClient, type SupabaseClient } from '@supabase/supabase-js';

// Sign-in is on when a Supabase project is configured: at runtime by the
// server's /config.js (single-instance deployment), or at build time via
// VITE_* (local development). Without either there is no login.
declare global {
  interface Window {
    __UPSTONK_CONFIG__?: { supabaseUrl?: string; supabasePublishableKey?: string };
  }
}
const runtime = typeof window !== 'undefined' ? window.__UPSTONK_CONFIG__ : undefined;
const url = runtime?.supabaseUrl || (import.meta.env.VITE_SUPABASE_URL as string | undefined);
const key = runtime?.supabasePublishableKey || (import.meta.env.VITE_SUPABASE_PUBLISHABLE_KEY as string | undefined);

export const supabase: SupabaseClient | null = url && key ? createClient(url, key) : null;
export const authEnabled = supabase !== null;
