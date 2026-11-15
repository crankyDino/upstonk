import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft } from 'lucide-react';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { toast } from 'sonner';
import { apiFetch } from '@/lib/api/client';
import { useAuth } from '@/auth/AuthProvider';

interface AdminUser {
  id: string;
  email: string;
  role: 'admin' | 'member';
  status: 'active' | 'disabled';
  invitedByEmail: string | null;
  createdAt: string;
  lastSeenAt: string | null;
}
interface AdminInvitation {
  id: number;
  email: string;
  invitedByEmail: string;
  createdAt: string;
  expiresAt: string;
  usedAt: string | null;
  revokedAt: string | null;
}
interface Settings {
  inviteOnly: boolean;
  notice: string | null;
  updatedAt: string;
}
interface DataStatus {
  listings: { instrumentType: string; active: number; inactive: number; lastImport: string | null; source: string | null }[];
  prices: { etfsWithPrices: number; latestDay: string | null; lastFetch: string | null; staleOrMissing: string[] };
  factsheets: { issuer: string; etfs: number; withFactsheet: number; oldest: string | null; newest: string | null }[];
  runs: { kind: string; startedAt: string; finishedAt: string | null; stored: number; failed: Record<string, string> | null }[];
}

const fmtDate = (s: string | null) => (s ? new Date(s).toLocaleString() : '—');
const errMessage = (e: unknown) => (e as { message?: string }).message ?? 'Something went wrong';

function UsersTab() {
  const qc = useQueryClient();
  const { me } = useAuth();
  const { data: users = [], isLoading } = useQuery({ queryKey: ['admin', 'users'], queryFn: () => apiFetch<AdminUser[]>('/admin/users') });
  const patch = useMutation({
    mutationFn: ({ id, body }: { id: string; body: Partial<Pick<AdminUser, 'role' | 'status'>> }) =>
      apiFetch<AdminUser[]>(`/admin/users/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
    onSuccess: (list) => qc.setQueryData(['admin', 'users'], list),
    onError: (e) => toast.error(errMessage(e)),
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Users</CardTitle>
        <CardDescription>Everyone who has been admitted. Addresses in ADMIN_EMAILS are always admins.</CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? <p className="text-muted-foreground">Loading…</p> : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Email</TableHead><TableHead>Role</TableHead><TableHead>Status</TableHead>
                <TableHead>Invited by</TableHead><TableHead>Last seen</TableHead><TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.map((u) => (
                <TableRow key={u.id}>
                  <TableCell>{u.email}{u.id === me?.id && <span className="text-muted-foreground"> (you)</span>}</TableCell>
                  <TableCell><Badge variant={u.role === 'admin' ? 'default' : 'secondary'}>{u.role}</Badge></TableCell>
                  <TableCell><Badge variant={u.status === 'active' ? 'outline' : 'destructive'}>{u.status}</Badge></TableCell>
                  <TableCell>{u.invitedByEmail ?? '—'}</TableCell>
                  <TableCell>{fmtDate(u.lastSeenAt)}</TableCell>
                  <TableCell className="text-right space-x-2">
                    <Button size="sm" variant="outline" disabled={patch.isPending}
                      onClick={() => patch.mutate({ id: u.id, body: { role: u.role === 'admin' ? 'member' : 'admin' } })}>
                      {u.role === 'admin' ? 'Make member' : 'Make admin'}
                    </Button>
                    <Button size="sm" variant={u.status === 'active' ? 'destructive' : 'outline'} disabled={patch.isPending}
                      onClick={() => patch.mutate({ id: u.id, body: { status: u.status === 'active' ? 'disabled' : 'active' } })}>
                      {u.status === 'active' ? 'Disable' : 'Enable'}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}

function invitationState(i: AdminInvitation): { label: string; variant: 'default' | 'secondary' | 'outline' | 'destructive' } {
  if (i.usedAt) return { label: 'used', variant: 'secondary' };
  if (i.revokedAt) return { label: 'revoked', variant: 'destructive' };
  if (new Date(i.expiresAt) < new Date()) return { label: 'expired', variant: 'outline' };
  return { label: 'open', variant: 'default' };
}

function InvitationsTab() {
  const qc = useQueryClient();
  const { data: invitations = [], isLoading } = useQuery({
    queryKey: ['admin', 'invitations'],
    queryFn: () => apiFetch<AdminInvitation[]>('/admin/invitations'),
  });
  const revoke = useMutation({
    mutationFn: (id: number) => apiFetch<void>(`/admin/invitations/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'invitations'] }),
    onError: (e) => toast.error(errMessage(e)),
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Invitations</CardTitle>
        <CardDescription>Links created by any member. Use "Invite" in the header to create one.</CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? <p className="text-muted-foreground">Loading…</p> : invitations.length === 0 ? (
          <p className="text-muted-foreground">No invitations yet.</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Email</TableHead><TableHead>Invited by</TableHead><TableHead>Created</TableHead>
                <TableHead>Expires</TableHead><TableHead>State</TableHead><TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {invitations.map((i) => {
                const st = invitationState(i);
                return (
                  <TableRow key={i.id}>
                    <TableCell>{i.email}</TableCell>
                    <TableCell>{i.invitedByEmail}</TableCell>
                    <TableCell>{fmtDate(i.createdAt)}</TableCell>
                    <TableCell>{fmtDate(i.expiresAt)}</TableCell>
                    <TableCell><Badge variant={st.variant}>{st.label}</Badge></TableCell>
                    <TableCell className="text-right">
                      {st.label === 'open' && (
                        <Button size="sm" variant="outline" disabled={revoke.isPending} onClick={() => revoke.mutate(i.id)}>Revoke</Button>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}

function SettingsTab() {
  const qc = useQueryClient();
  const { data, isLoading } = useQuery({ queryKey: ['admin', 'settings'], queryFn: () => apiFetch<Settings>('/admin/settings') });
  const [inviteOnly, setInviteOnly] = useState(true);
  const [notice, setNotice] = useState('');
  useEffect(() => {
    if (data) {
      setInviteOnly(data.inviteOnly);
      setNotice(data.notice ?? '');
    }
  }, [data]);
  const save = useMutation({
    mutationFn: () => apiFetch<Settings>('/admin/settings', { method: 'PUT', body: JSON.stringify({ inviteOnly, notice: notice.trim() || null }) }),
    onSuccess: (s) => {
      qc.setQueryData(['admin', 'settings'], s);
      qc.invalidateQueries({ queryKey: ['notice'] });
      toast.success('Settings saved');
    },
    onError: (e) => toast.error(errMessage(e)),
  });

  if (isLoading) return <p className="text-muted-foreground">Loading…</p>;
  return (
    <Card>
      <CardHeader>
        <CardTitle>App settings</CardTitle>
        <CardDescription>Last changed {fmtDate(data?.updatedAt ?? null)}.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="flex items-start justify-between gap-4 p-4 border rounded-lg">
          <div className="space-y-1">
            <Label htmlFor="invite-only" className="text-base">Invite-only</Label>
            <p className="text-sm text-muted-foreground">
              On: new accounts need an invite link from a member. Off: anyone who signs up gets in. Keep this on;
              the data sources' terms only cover a small, personal circle of users.
            </p>
          </div>
          <Switch id="invite-only" checked={inviteOnly} onCheckedChange={setInviteOnly} />
        </div>
        <div className="space-y-2">
          <Label htmlFor="notice">Notice shown to everyone</Label>
          <Textarea id="notice" placeholder="e.g. Factsheet data refreshed on the 1st of each month" value={notice}
            onChange={(e) => setNotice(e.target.value)} />
        </div>
        <Button onClick={() => save.mutate()} disabled={save.isPending}>Save settings</Button>
      </CardContent>
    </Card>
  );
}

function DataStatusTab() {
  const { data, isLoading } = useQuery({ queryKey: ['admin', 'data-status'], queryFn: () => apiFetch<DataStatus>('/admin/data-status') });
  if (isLoading || !data) return <p className="text-muted-foreground">Loading…</p>;
  return (
    <div className="space-y-6">
      <div className="grid gap-4 md:grid-cols-3">
        {data.listings.map((l) => (
          <Card key={l.instrumentType}>
            <CardHeader className="pb-2"><CardDescription>{l.instrumentType}s listed</CardDescription><CardTitle className="text-3xl">{l.active}</CardTitle></CardHeader>
            <CardContent className="text-sm text-muted-foreground">{l.source ?? '—'}, imported {l.lastImport ?? '—'}{l.inactive > 0 && `; ${l.inactive} delisted`}</CardContent>
          </Card>
        ))}
        <Card>
          <CardHeader className="pb-2"><CardDescription>ETFs with prices</CardDescription><CardTitle className="text-3xl">{data.prices.etfsWithPrices}</CardTitle></CardHeader>
          <CardContent className="text-sm text-muted-foreground">Latest trading day {data.prices.latestDay ?? '—'}; fetched {fmtDate(data.prices.lastFetch)}</CardContent>
        </Card>
      </div>

      {data.prices.staleOrMissing.length > 0 && (
        <Card>
          <CardHeader><CardTitle className="text-base">Stale or missing prices ({data.prices.staleOrMissing.length})</CardTitle>
            <CardDescription>Active ETFs with no price within 7 days of the newest stored day.</CardDescription></CardHeader>
          <CardContent className="text-sm">{data.prices.staleOrMissing.join(', ')}</CardContent>
        </Card>
      )}

      <Card>
        <CardHeader><CardTitle className="text-base">Factsheet coverage by issuer</CardTitle></CardHeader>
        <CardContent>
          <Table>
            <TableHeader><TableRow><TableHead>Issuer</TableHead><TableHead>ETFs</TableHead><TableHead>With factsheet</TableHead><TableHead>Oldest</TableHead><TableHead>Newest</TableHead></TableRow></TableHeader>
            <TableBody>
              {data.factsheets.map((f) => (
                <TableRow key={f.issuer}>
                  <TableCell>{f.issuer}</TableCell><TableCell>{f.etfs}</TableCell><TableCell>{f.withFactsheet}</TableCell>
                  <TableCell>{f.oldest ?? '—'}</TableCell><TableCell>{f.newest ?? '—'}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle className="text-base">Last ingest runs</CardTitle>
          <CardDescription>Runs are started with <code>go run ./cmd/ingest …</code> on the admin's machine.</CardDescription></CardHeader>
        <CardContent>
          {data.runs.length === 0 ? <p className="text-muted-foreground">No runs recorded yet.</p> : (
            <Table>
              <TableHeader><TableRow><TableHead>Job</TableHead><TableHead>Started</TableHead><TableHead>Finished</TableHead><TableHead>Stored</TableHead><TableHead>Failures</TableHead></TableRow></TableHeader>
              <TableBody>
                {data.runs.map((r) => {
                  const failures = Object.entries(r.failed ?? {});
                  return (
                    <TableRow key={r.kind}>
                      <TableCell>{r.kind}</TableCell>
                      <TableCell>{fmtDate(r.startedAt)}</TableCell>
                      <TableCell>{r.finishedAt ? fmtDate(r.finishedAt) : <Badge variant="destructive">did not finish</Badge>}</TableCell>
                      <TableCell>{r.stored}</TableCell>
                      <TableCell className="text-sm">
                        {failures.length === 0 ? 'none' : failures.map(([code, why]) => <div key={code}><strong>{code}</strong>: {why}</div>)}
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

export default function Admin() {
  return (
    <div className="min-h-screen bg-background">
      <header className="border-b bg-card/50">
        <div className="container mx-auto px-4 h-16 flex items-center gap-3">
          <Button variant="ghost" size="sm" asChild><Link to="/"><ArrowLeft className="h-4 w-4 mr-2" />Back</Link></Button>
          <h1 className="font-bold text-xl">Admin</h1>
        </div>
      </header>
      <main className="container mx-auto px-4 py-6">
        <Tabs defaultValue="users">
          <TabsList className="flex-wrap h-auto">
            <TabsTrigger value="users">Users</TabsTrigger>
            <TabsTrigger value="invitations">Invitations</TabsTrigger>
            <TabsTrigger value="settings">Settings</TabsTrigger>
            <TabsTrigger value="data">Data status</TabsTrigger>
          </TabsList>
          <TabsContent value="users" className="mt-4"><UsersTab /></TabsContent>
          <TabsContent value="invitations" className="mt-4"><InvitationsTab /></TabsContent>
          <TabsContent value="settings" className="mt-4"><SettingsTab /></TabsContent>
          <TabsContent value="data" className="mt-4"><DataStatusTab /></TabsContent>
        </Tabs>
      </main>
    </div>
  );
}
