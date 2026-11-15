import { useQuery } from '@tanstack/react-query';
import { Info } from 'lucide-react';
import { apiFetch } from '@/lib/api/client';
import { useAuth } from '@/auth/AuthProvider';

// The app-wide notice set by an admin.
export function NoticeBanner() {
  const { enabled, me } = useAuth();
  const { data } = useQuery({
    queryKey: ['notice'],
    queryFn: () => apiFetch<{ notice: string | null }>('/notice'),
    enabled: enabled && !!me,
    staleTime: 5 * 60 * 1000,
  });
  if (!data?.notice) return null;
  return (
    <div className="bg-primary/10 border-b text-sm">
      <div className="container mx-auto px-4 py-2 flex items-center gap-2">
        <Info className="h-4 w-4 shrink-0" />
        <span>{data.notice}</span>
      </div>
    </div>
  );
}
