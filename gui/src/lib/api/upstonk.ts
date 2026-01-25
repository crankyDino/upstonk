import type { DiscoveryRequest, DiscoveryResponse, APIError } from '@/types/api';

const API_BASE_URL = import.meta.env.VITE_UPSTONK_API_URL || 'http://localhost:8080/api/v1';

export async function discoverETFs(request: DiscoveryRequest): Promise<DiscoveryResponse> {
  const response = await fetch(`${API_BASE_URL}/discover`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw {
      code: errorData.code || 'API_ERROR',
      message: errorData.message || `Request failed with status ${response.status}`,
      requestId: errorData.requestId || `req_${Date.now()}`,
    } as APIError;
  }

  return response.json();
}

export function formatCurrency(value: number, currency: string = 'USD'): string {
  if (value >= 1e12) return `$${(value / 1e12).toFixed(1)}T`;
  if (value >= 1e9) return `$${(value / 1e9).toFixed(1)}B`;
  if (value >= 1e6) return `$${(value / 1e6).toFixed(1)}M`;
  return new Intl.NumberFormat('en-US', { style: 'currency', currency }).format(value);
}

export function formatPercentage(value: number): string {
  return `${(value * 100).toFixed(2)}%`;
}
