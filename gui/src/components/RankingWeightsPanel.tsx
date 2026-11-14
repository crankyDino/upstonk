import { useDiscoveryStore } from '@/stores/discoveryStore';
import { Label } from '@/components/ui/label';
import { Slider } from '@/components/ui/slider';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Scale } from 'lucide-react';
import { defaultRankingPreferences } from '@/lib/discoveryOptions';
import type { RankingComponent } from '@/types/api';

const HINTS: Record<RankingComponent, string> = {
  fees: 'Lower TER scores higher',
  liquidity: 'Higher average daily volume scores higher',
  tracking: 'Smaller tracking difference scores higher',
  stability: 'Larger fund size (AUM) scores higher',
};

// Each slider sets a raw 0–100 weight. The request builder normalises them to
// sum to 1, so the share shown here is what the server will actually use.
export function RankingWeightsPanel() {
  const { rankingPreferences, setRankingPreferences } = useDiscoveryStore();
  const total = rankingPreferences.reduce((sum, p) => sum + Math.max(0, p.weight), 0);

  const setWeight = (id: RankingComponent, percent: number) =>
    setRankingPreferences(
      rankingPreferences.map((p) => (p.id === id ? { ...p, weight: percent / 100 } : p)),
    );

  const share = (weight: number) =>
    total > 0 ? Math.round((Math.max(0, weight) / total) * 100) : Math.round(100 / rankingPreferences.length);

  return (
    <Card className="border-border/50 shadow-sm mt-6">
      <CardHeader className="pb-4">
        <CardTitle className="flex items-center gap-2 text-lg">
          <Scale className="h-5 w-5 text-primary" />
          Ranking Weights
        </CardTitle>
        <CardDescription>
          How much each quality factor counts when ordering results. Shares always add up to 100%.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="grid gap-6 md:grid-cols-2">
          {rankingPreferences.map((pref) => (
            <div key={pref.id} className="space-y-3 p-4 border rounded-lg">
              <div className="flex items-baseline justify-between gap-2">
                <Label>{pref.name}</Label>
                <span className="text-sm font-medium tabular-nums">{share(pref.weight)}%</span>
              </div>
              <Slider
                value={[Math.round(pref.weight * 100)]}
                onValueChange={([value]) => setWeight(pref.id, value)}
                min={0}
                max={100}
                step={5}
                aria-label={`${pref.name} weight`}
              />
              <p className="text-xs text-muted-foreground">{HINTS[pref.id]}</p>
            </div>
          ))}
        </div>
        {total === 0 && (
          <p className="text-sm text-muted-foreground">All weights are zero, so every factor counts equally.</p>
        )}
        <Button variant="outline" size="sm" onClick={() => setRankingPreferences(defaultRankingPreferences)}>
          Reset to defaults
        </Button>
      </CardContent>
    </Card>
  );
}
