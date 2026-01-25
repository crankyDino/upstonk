import { useDiscoveryStore } from '@/stores/discoveryStore';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Switch } from '@/components/ui/switch';
import { Globe, Building2, BarChart3, MapPin, X } from 'lucide-react';
import { useState } from 'react';

const assetClasses = [
  { id: 'equity', label: 'Equity', description: 'Stocks and equity funds' },
  { id: 'fixed_income', label: 'Fixed Income', description: 'Bonds and bond funds' },
  { id: 'commodities', label: 'Commodities', description: 'Gold, oil, and other commodities' },
  { id: 'real_estate', label: 'Real Estate', description: 'REITs and property funds' },
  { id: 'alternatives', label: 'Alternatives', description: 'Hedge funds, private equity' },
];

const sectors = [
  'Technology', 'Healthcare', 'Financial', 'Consumer Discretionary', 
  'Consumer Staples', 'Energy', 'Industrials', 'Materials', 
  'Real Estate', 'Utilities', 'Communications', 'Clean Energy',
  'Semiconductors', 'Biotechnology', 'Cybersecurity'
];

const markets = [
  { code: 'US', name: 'United States' },
  { code: 'CA', name: 'Canada' },
  { code: 'GB', name: 'United Kingdom' },
  { code: 'EU', name: 'Europe' },
  { code: 'JP', name: 'Japan' },
  { code: 'CN', name: 'China' },
  { code: 'GLOBAL', name: 'Global' },
];

export function ExposureSelector() {
  const { exposure, setExposure } = useDiscoveryStore();
  const [sectorSearch, setSectorSearch] = useState('');

  const toggleAssetClass = (id: string) => {
    const newClasses = exposure.assetClasses.includes(id)
      ? exposure.assetClasses.filter(c => c !== id)
      : [...exposure.assetClasses, id];
    setExposure({ assetClasses: newClasses });
  };

  const toggleSector = (sector: string) => {
    const newSectors = exposure.sectors.includes(sector)
      ? exposure.sectors.filter(s => s !== sector)
      : [...exposure.sectors, sector];
    setExposure({ sectors: newSectors });
  };

  const toggleMarket = (code: string) => {
    const newMarkets = exposure.geography.markets.includes(code)
      ? exposure.geography.markets.filter(m => m !== code)
      : [...exposure.geography.markets, code];
    setExposure({ 
      geography: { ...exposure.geography, markets: newMarkets }
    });
  };

  const filteredSectors = sectors.filter(s => 
    s.toLowerCase().includes(sectorSearch.toLowerCase())
  );

  return (
    <Card className="border-border/50 shadow-sm">
      <CardHeader className="pb-4">
        <CardTitle className="flex items-center gap-2 text-lg">
          <Globe className="h-5 w-5 text-primary" />
          Exposure Configuration
        </CardTitle>
        <CardDescription>
          Define your desired investment exposure and geographic preferences
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Asset Classes */}
        <div className="space-y-3">
          <Label className="flex items-center gap-2">
            <BarChart3 className="h-4 w-4 text-muted-foreground" />
            Asset Classes
          </Label>
          <div className="flex flex-wrap gap-2">
            {assetClasses.map((asset) => (
              <Badge
                key={asset.id}
                variant={exposure.assetClasses.includes(asset.id) ? 'default' : 'outline'}
                className={`cursor-pointer transition-all ${
                  exposure.assetClasses.includes(asset.id) 
                    ? 'bg-primary text-primary-foreground' 
                    : 'hover:bg-accent'
                }`}
                onClick={() => toggleAssetClass(asset.id)}
              >
                {asset.label}
              </Badge>
            ))}
          </div>
        </div>

        {/* Sectors */}
        <div className="space-y-3">
          <Label className="flex items-center gap-2">
            <Building2 className="h-4 w-4 text-muted-foreground" />
            Sectors ({exposure.sectors.length} selected)
          </Label>
          
          {exposure.sectors.length > 0 && (
            <div className="flex flex-wrap gap-1.5 p-3 border rounded-lg bg-muted/30">
              {exposure.sectors.map((sector) => (
                <Badge 
                  key={sector} 
                  variant="secondary"
                  className="gap-1 pr-1"
                >
                  {sector}
                  <X 
                    className="h-3 w-3 cursor-pointer hover:text-destructive" 
                    onClick={() => toggleSector(sector)}
                  />
                </Badge>
              ))}
            </div>
          )}
          
          <input
            type="text"
            placeholder="Search sectors..."
            value={sectorSearch}
            onChange={(e) => setSectorSearch(e.target.value)}
            className="w-full px-3 py-2 text-sm border rounded-lg bg-background focus:outline-none focus:ring-2 focus:ring-ring"
          />
          
          <div className="flex flex-wrap gap-1.5 max-h-32 overflow-y-auto p-2 border rounded-lg">
            {filteredSectors.map((sector) => (
              <Badge
                key={sector}
                variant={exposure.sectors.includes(sector) ? 'default' : 'outline'}
                className={`cursor-pointer text-xs transition-all ${
                  exposure.sectors.includes(sector) 
                    ? 'bg-primary text-primary-foreground' 
                    : 'hover:bg-accent'
                }`}
                onClick={() => toggleSector(sector)}
              >
                {sector}
              </Badge>
            ))}
          </div>
        </div>

        {/* Geography */}
        <div className="space-y-4">
          <Label className="flex items-center gap-2">
            <MapPin className="h-4 w-4 text-muted-foreground" />
            Geographic Focus
          </Label>
          
          <div className="flex flex-wrap gap-2">
            {markets.map((market) => (
              <Badge
                key={market.code}
                variant={exposure.geography.markets.includes(market.code) ? 'default' : 'outline'}
                className={`cursor-pointer transition-all ${
                  exposure.geography.markets.includes(market.code) 
                    ? 'bg-primary text-primary-foreground' 
                    : 'hover:bg-accent'
                }`}
                onClick={() => toggleMarket(market.code)}
              >
                {market.name}
              </Badge>
            ))}
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <div className="flex items-center justify-between p-3 border rounded-lg">
              <Label className="font-normal">Include Emerging Markets</Label>
              <Switch
                checked={exposure.geography.includeEmerging}
                onCheckedChange={(checked) => 
                  setExposure({ 
                    geography: { ...exposure.geography, includeEmerging: checked }
                  })
                }
              />
            </div>
            
            <div className="flex items-center justify-between p-3 border rounded-lg">
              <Label className="font-normal">Include Developed Markets</Label>
              <Switch
                checked={exposure.geography.includeDeveloped}
                onCheckedChange={(checked) => 
                  setExposure({ 
                    geography: { ...exposure.geography, includeDeveloped: checked }
                  })
                }
              />
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
