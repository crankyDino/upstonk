import { useDiscoveryStore } from '@/stores/discoveryStore';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Slider } from '@/components/ui/slider';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Settings2, Ban, Percent, DollarSign, Droplets } from 'lucide-react';
import { formatCurrency } from '@/lib/api/upstonk';

export function ConstraintsPanel() {
  const { constraints, setConstraints } = useDiscoveryStore();

  return (
    <Card className="border-border/50 shadow-sm">
      <CardHeader className="pb-4">
        <CardTitle className="flex items-center gap-2 text-lg">
          <Settings2 className="h-5 w-5 text-primary" />
          Constraints & Filters
        </CardTitle>
        <CardDescription>
          Set investment constraints and screening criteria
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Eligibility Constraint */}
        <div className="flex items-center justify-between p-4 border rounded-lg bg-card">
          <div className="space-y-0.5">
            <Label className="text-base font-medium">TFSA-Eligible Only</Label>
            <p className="text-sm text-muted-foreground">
              Only show ETFs confirmed eligible for your account type
            </p>
          </div>
          <Switch
            checked={constraints.tfsaEligibleOnly}
            onCheckedChange={(checked) => setConstraints({ tfsaEligibleOnly: checked })}
          />
        </div>

        {/* Numeric Constraints */}
        <div className="grid gap-6 md:grid-cols-2">
          {/* Max TER */}
          <div className="space-y-4 p-4 border rounded-lg">
            <Label className="flex items-center gap-2">
              <Percent className="h-4 w-4 text-muted-foreground" />
              Max Expense Ratio (TER): {(constraints.maxTER * 100).toFixed(2)}%
            </Label>
            <Slider
              value={[constraints.maxTER * 100]}
              onValueChange={([value]) => setConstraints({ maxTER: value / 100 })}
              min={0}
              max={100}
              step={1}
              className="w-full"
            />
            <div className="flex justify-between text-xs text-muted-foreground">
              <span>0%</span>
              <span>0.5%</span>
              <span>1%</span>
            </div>
          </div>

          {/* Min AUM */}
          <div className="space-y-4 p-4 border rounded-lg">
            <Label className="flex items-center gap-2">
              <DollarSign className="h-4 w-4 text-muted-foreground" />
              Min AUM: {formatCurrency(constraints.minAUM)}
            </Label>
            <Slider
              value={[Math.log10(constraints.minAUM)]}
              onValueChange={([value]) => setConstraints({ minAUM: Math.pow(10, value) })}
              min={6}
              max={12}
              step={0.1}
              className="w-full"
            />
            <div className="flex justify-between text-xs text-muted-foreground">
              <span>$1M</span>
              <span>$1B</span>
              <span>$1T</span>
            </div>
          </div>

          {/* Liquidity Threshold */}
          <div className="space-y-4 p-4 border rounded-lg md:col-span-2">
            <Label className="flex items-center gap-2">
              <Droplets className="h-4 w-4 text-muted-foreground" />
              Minimum Liquidity Score: {constraints.liquidityThreshold}%
            </Label>
            <Slider
              value={[constraints.liquidityThreshold]}
              onValueChange={([value]) => setConstraints({ liquidityThreshold: value })}
              min={0}
              max={100}
              step={5}
              className="w-full"
            />
            <div className="flex justify-between text-xs text-muted-foreground">
              <span>Low</span>
              <span>Medium</span>
              <span>High</span>
            </div>
          </div>
        </div>

        {/* Exclusions */}
        <div className="space-y-4">
          <div className="flex items-center gap-2 mb-2">
            <Ban className="h-4 w-4 text-muted-foreground" />
            <Label className="text-base font-medium">Exclusions</Label>
          </div>
          
          <div className="grid gap-4 md:grid-cols-2">
            <div className="flex items-center justify-between p-3 border rounded-lg">
              <Label className="font-normal">Synthetic ETFs</Label>
              <Switch
                checked={constraints.excludeSynthetic}
                onCheckedChange={(checked) => setConstraints({ excludeSynthetic: checked })}
              />
            </div>
            
            <div className="flex items-center justify-between p-3 border rounded-lg">
              <Label className="font-normal">Leveraged ETFs</Label>
              <Switch
                checked={constraints.excludeLeveraged}
                onCheckedChange={(checked) => setConstraints({ excludeLeveraged: checked })}
              />
            </div>
            
            <div className="flex items-center justify-between p-3 border rounded-lg">
              <Label className="font-normal">Inverse ETFs</Label>
              <Switch
                checked={constraints.excludeInverse}
                onCheckedChange={(checked) => setConstraints({ excludeInverse: checked })}
              />
            </div>
            
            <div className="flex items-center justify-between p-3 border rounded-lg">
              <Label className="font-normal">Physical Replication Only</Label>
              <Switch
                checked={constraints.physicalReplicationOnly}
                onCheckedChange={(checked) => setConstraints({ physicalReplicationOnly: checked })}
              />
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
