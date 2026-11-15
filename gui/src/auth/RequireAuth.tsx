import type { ReactNode } from 'react';
import { Button } from '@/components/ui/button';
import { useAuth } from '@/auth/AuthProvider';
import { SignInCard } from '@/auth/SignInCard';

function Centered({ children }: { children: ReactNode }) {
  return <div className="min-h-screen flex items-center justify-center bg-background px-4">{children}</div>;
}

// Renders children only for an admitted member (or always when sign-in
// isn't configured).
export function RequireAuth({ children, adminOnly = false }: { children: ReactNode; adminOnly?: boolean }) {
  const { enabled, loading, session, me, refusal, signOut } = useAuth();
  if (!enabled) return <>{children}</>;
  if (loading) return <Centered><p className="text-muted-foreground">Loading…</p></Centered>;
  if (!session) return <Centered><SignInCard /></Centered>;
  if (refusal || !me) {
    return (
      <Centered>
        <div className="max-w-sm text-center space-y-4">
          <h2 className="text-xl font-semibold">
            {refusal?.code === 'ACCOUNT_DISABLED' ? 'Account disabled' : 'Invite needed'}
          </h2>
          <p className="text-muted-foreground">{refusal?.message ?? 'Your account is not set up yet.'}</p>
          <Button variant="outline" onClick={signOut}>Sign out</Button>
        </div>
      </Centered>
    );
  }
  if (adminOnly && me.role !== 'admin') {
    return <Centered><p className="text-muted-foreground">Admins only.</p></Centered>;
  }
  return <>{children}</>;
}
