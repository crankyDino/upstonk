import type { DiscoveryRequest, DiscoveryResponse } from '@/types/api';
import { apiFetch } from '@/lib/api/client';

export function discoverETFs(request: DiscoveryRequest): Promise<DiscoveryResponse> {
  return apiFetch<DiscoveryResponse>('/discover', { method: 'POST', body: JSON.stringify(request) });
}

const SYMBOLS: Record<string, string> = { ZAR: 'R', USD: '$', GBP: '£', EUR: '€' };

// Fund size: 0 means no source reported it, shown as "—", never as a value.
export function formatCurrency(value: number, currency: string = 'ZAR'): string {
  if (!value) return '—';
  const s = SYMBOLS[currency] ?? `${currency} `;
  if (value >= 1e12) return `${s}${(value / 1e12).toFixed(1)}T`;
  if (value >= 1e9) return `${s}${(value / 1e9).toFixed(1)}bn`;
  if (value >= 1e6) return `${s}${(value / 1e6).toFixed(1)}m`;
  return `${s}${value.toLocaleString('en-ZA', { maximumFractionDigits: 0 })}`;
}

// TER arrives already in percent (0.10 = 0.10%); 0 means not reported.
export function formatTER(terPercent: number): string {
  if (!terPercent) return '—';
  return `${terPercent.toFixed(2)}%`;
}

// A fraction (0.25) as a percentage (25.00%).
export function formatPercentage(value: number): string {
  return `${(value * 100).toFixed(2)}%`;
}
