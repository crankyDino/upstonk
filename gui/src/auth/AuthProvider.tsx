import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react';
import type { Session } from '@supabase/supabase-js';
import { authEnabled, supabase } from '@/lib/supabase';
import { apiFetch, forgetInviteToken } from '@/lib/api/client';

export interface Me {
  id: string;
  email: string;
  role: 'admin' | 'member';
}

interface AuthState {
  enabled: boolean;
  loading: boolean;
  session: Session | null;
  me: Me | null;
  // Set when the server refused the signed-in user (e.g. NOT_INVITED).
  refusal: { code: string; message: string } | null;
  signIn: (email: string, password: string) => Promise<string | null>;
  signUp: (email: string, password: string) => Promise<{ error: string | null; needsConfirmation: boolean }>;
  signOut: () => Promise<void>;
  refresh: () => Promise<void>;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(null);
  const [me, setMe] = useState<Me | null>(null);
  const [refusal, setRefusal] = useState<AuthState['refusal']>(null);
  const [loading, setLoading] = useState(authEnabled);

  const loadMe = useCallback(async (s: Session | null) => {
    if (!s) {
      setMe(null);
      setRefusal(null);
      setLoading(false);
      return;
    }
    try {
      const m = await apiFetch<Me>('/me');
      setMe(m);
      setRefusal(null);
      forgetInviteToken(); // admitted: the invite link has done its job
    } catch (e) {
      const err = e as { code?: string; message?: string };
      setMe(null);
      setRefusal({ code: err.code ?? 'ERROR', message: err.message ?? 'Could not load your account' });
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!supabase) return;
    supabase.auth.getSession().then(({ data }) => {
      setSession(data.session);
      loadMe(data.session);
    });
    const { data: sub } = supabase.auth.onAuthStateChange((_event, s) => {
      setSession(s);
      loadMe(s);
    });
    return () => sub.subscription.unsubscribe();
  }, [loadMe]);

  const value: AuthState = {
    enabled: authEnabled,
    loading,
    session,
    me,
    refusal,
    signIn: async (email, password) => {
      if (!supabase) return 'Sign-in is not configured';
      const { error } = await supabase.auth.signInWithPassword({ email, password });
      return error ? error.message : null;
    },
    signUp: async (email, password) => {
      if (!supabase) return { error: 'Sign-in is not configured', needsConfirmation: false };
      const { data, error } = await supabase.auth.signUp({
        email,
        password,
        options: { emailRedirectTo: window.location.origin },
      });
      return { error: error ? error.message : null, needsConfirmation: !error && !data.session };
    },
    signOut: async () => {
      await supabase?.auth.signOut();
      setMe(null);
    },
    refresh: async () => loadMe(session),
  };
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider');
  return ctx;
}
