import { useEffect } from 'react';
import { Navigate, useParams } from 'react-router-dom';
import { rememberInviteToken } from '@/lib/api/client';
import { useAuth } from '@/auth/AuthProvider';
import { SignInCard } from '@/auth/SignInCard';

// /invite/:token: keeps the token (it's sent with requests until the server
// admits you) and offers account creation.
export default function Invite() {
  const { token } = useParams();
  const { session, me, refresh } = useAuth();

  useEffect(() => {
    if (!token) return;
    rememberInviteToken(token);
    if (session && !me) refresh(); // already signed in: try again with the invite
  }, [token, session, me, refresh]);

  if (session && me) return <Navigate to="/" replace />;
  return (
    <div className="min-h-screen flex flex-col items-center justify-center gap-6 bg-background px-4">
      <div className="text-center space-y-1">
        <h1 className="text-2xl font-bold">You're invited to UPSTONK</h1>
        <p className="text-muted-foreground">Create an account with the email address you were invited with.</p>
      </div>
      <SignInCard startWithSignUp />
    </div>
  );
}
