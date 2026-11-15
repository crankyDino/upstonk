import { createClient, type SupabaseClient } from '@supabase/supabase-js';

// Sign-in is on when the Supabase project is configured at build time.
// Without it (local development against an open API) there is no login.
const url = import.meta.env.VITE_SUPABASE_URL as string | undefined;
const key = import.meta.env.VITE_SUPABASE_PUBLISHABLE_KEY as string | undefined;

export const supabase: SupabaseClient | null = url && key ? createClient(url, key) : null;
export const authEnabled = supabase !== null;
