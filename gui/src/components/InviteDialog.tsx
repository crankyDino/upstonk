import { useState, type FormEvent } from 'react';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { UserPlus, Copy, Check } from 'lucide-react';
import { apiFetch } from '@/lib/api/client';

interface Invitation {
  email: string;
  link: string;
  expiresAt: string;
}

// Any member can invite: creates a one-time link to share however you like.
export function InviteDialog() {
  const [email, setEmail] = useState('');
  const [invitation, setInvitation] = useState<Invitation | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const create = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    try {
      setInvitation(await apiFetch<Invitation>('/invitations', { method: 'POST', body: JSON.stringify({ email }) }));
    } catch (err) {
      setError((err as { message?: string }).message ?? 'Could not create the invitation');
    }
  };
  const fullLink = invitation ? `${window.location.origin}${invitation.link}` : '';

  return (
    <Dialog onOpenChange={(open) => !open && (setInvitation(null), setEmail(''), setCopied(false))}>
      <DialogTrigger asChild>
        <Button variant="ghost" size="sm"><UserPlus className="h-4 w-4 mr-2" />Invite</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Invite someone</DialogTitle>
          <DialogDescription>They can join using the link with this email address. Links work once and expire after 14 days.</DialogDescription>
        </DialogHeader>
        {!invitation ? (
          <form onSubmit={create} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="invite-email">Their email</Label>
              <Input id="invite-email" type="email" required value={email} onChange={(e) => setEmail(e.target.value)} />
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
            <Button type="submit">Create invite link</Button>
          </form>
        ) : (
          <div className="space-y-3">
            <p className="text-sm">Send this link to <strong>{invitation.email}</strong>:</p>
            <div className="flex gap-2">
              <Input readOnly value={fullLink} onFocus={(e) => e.currentTarget.select()} />
              <Button
                variant="outline"
                size="icon"
                aria-label="Copy link"
                onClick={async () => {
                  try {
                    await navigator.clipboard.writeText(fullLink);
                    setCopied(true);
                  } catch {
                    /* select-and-copy by hand */
                  }
                }}
              >
                {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">Expires {new Date(invitation.expiresAt).toLocaleDateString()}.</p>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
