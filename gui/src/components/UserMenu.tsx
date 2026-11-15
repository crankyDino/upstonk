import { Link } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { LogOut, Shield } from 'lucide-react';
import { useAuth } from '@/auth/AuthProvider';
import { InviteDialog } from '@/components/InviteDialog';

export function UserMenu() {
  const { enabled, me, signOut } = useAuth();
  if (!enabled || !me) return null;
  return (
    <div className="flex items-center gap-1">
      <InviteDialog />
      {me.role === 'admin' && (
        <Button variant="ghost" size="sm" asChild>
          <Link to="/admin"><Shield className="h-4 w-4 mr-2" />Admin</Link>
        </Button>
      )}
      <span className="hidden md:inline text-sm text-muted-foreground px-2">{me.email}</span>
      <Button variant="ghost" size="sm" onClick={signOut} aria-label="Sign out"><LogOut className="h-4 w-4" /></Button>
    </div>
  );
}
