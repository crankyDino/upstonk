import { useState } from 'react';
import { useDiscoveryStore } from '@/stores/discoveryStore';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from '@/components/ui/sheet';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { Button } from '@/components/ui/button';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { EligibilityBadge } from './EligibilityBadge';
import { formatCurrency, formatPercentage } from '@/lib/api/upstonk';
import { 
  ExternalLink, 
  CheckCircle2, 
  AlertTriangle, 
  Database, 
  TrendingUp, 
  ChevronDown,
  ChevronUp,
  PieChart,
  Building2
} from 'lucide-react';

export function ETFDetailDrawer() {
  const { response, selectedETFTicker, setSelectedETF } = useDiscoveryStore();
  const [holdingsExpanded, setHoldingsExpanded] = useState(false);
  
  const etf = response?.results.find(r => r.ticker === selectedETFTicker);
  const isOpen = !!selectedETFTicker && !!etf;

  const topHoldings = etf?.topHoldings || [];
  const displayedHoldings = holdingsExpanded ? topHoldings : topHoldings.slice(0, 5);
  const hasMoreHoldings = topHoldings.length > 5;

  return (
    <Sheet open={isOpen} onOpenChange={(open) => { 
      if (!open) {
        setSelectedETF(null);
        setHoldingsExpanded(false);
      }
    }}>
      <SheetContent className="w-full sm:max-w-xl overflow-y-auto">
        {etf && (
          <>
            <SheetHeader className="pb-4">
              <div className="flex items-start justify-between">
                <div>
                  <SheetTitle className="text-2xl font-bold">
                    <span className="font-mono text-primary">{etf.ticker}</span>
                  </SheetTitle>
                  <SheetDescription className="text-base mt-1">
                    {etf.name || 'ETF Details'}
                  </SheetDescription>
                </div>
                <EligibilityBadge 
                  status={etf.eligibility.status} 
                  confidence={etf.eligibility.confidence}
                  ruleVersion={etf.eligibility.ruleVersion}
                />
              </div>
            </SheetHeader>

            <div className="space-y-6">
              {/* Key Metrics */}
              <div className="grid grid-cols-2 gap-4">
                <div className="p-4 border rounded-lg bg-muted/20">
                  <div className="text-sm text-muted-foreground">Expense Ratio (TER)</div>
                  <div className="text-2xl font-bold">{formatPercentage(etf.ter)}</div>
                </div>
                <div className="p-4 border rounded-lg bg-muted/20">
                  <div className="text-sm text-muted-foreground">Assets Under Management</div>
                  <div className="text-2xl font-bold">{formatCurrency(etf.aum)}</div>
                </div>
                <div className="p-4 border rounded-lg bg-muted/20">
                  <div className="text-sm text-muted-foreground">Ranking Score</div>
                  <div className="text-2xl font-bold text-primary">{etf.rankingScore.toFixed(1)}</div>
                </div>
                <div className="p-4 border rounded-lg bg-muted/20">
                  <div className="text-sm text-muted-foreground">Match Score</div>
                  <div className="text-2xl font-bold">{etf.matchScore}%</div>
                </div>
              </div>

              <Separator />

              {/* ETF Details */}
              <div className="space-y-3">
                <h4 className="font-semibold flex items-center gap-2">
                  <TrendingUp className="h-4 w-4" />
                  Fund Details
                </h4>
                <div className="grid grid-cols-2 gap-3 text-sm">
                  {etf.provider && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Provider</span>
                      <span className="font-medium">{etf.provider}</span>
                    </div>
                  )}
                  {etf.currency && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Currency</span>
                      <span className="font-medium">{etf.currency}</span>
                    </div>
                  )}
                  {etf.isin && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">ISIN</span>
                      <span className="font-medium font-mono">{etf.isin}</span>
                    </div>
                  )}
                  {etf.exchange && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Exchange</span>
                      <span className="font-medium">{etf.exchange}</span>
                    </div>
                  )}
                  {etf.assetClass && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Asset Class</span>
                      <span className="font-medium">{etf.assetClass}</span>
                    </div>
                  )}
                  {etf.geographicFocus && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Geographic Focus</span>
                      <span className="font-medium">{etf.geographicFocus}</span>
                    </div>
                  )}
                </div>
                {etf.trackingIndex && (
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Tracking Index</span>
                    <span className="font-medium text-right max-w-[60%]">{etf.trackingIndex}</span>
                  </div>
                )}
              </div>

              <Separator />

              {/* Top Holdings */}
              {topHoldings.length > 0 && (
                <>
                  <div className="space-y-3">
                    <h4 className="font-semibold flex items-center gap-2">
                      <Building2 className="h-4 w-4" />
                      Top Holdings
                    </h4>
                    <div className="space-y-2">
                      {displayedHoldings.map((holding, i) => (
                        <div 
                          key={i} 
                          className="flex items-center justify-between p-3 border rounded-lg bg-muted/10"
                        >
                          <div className="flex items-center gap-3">
                            <span className="text-xs text-muted-foreground w-5">{i + 1}</span>
                            <div>
                              <div className="font-medium text-sm">{holding.name}</div>
                              {holding.ticker && holding.ticker !== 'n/a' && (
                                <div className="text-xs text-muted-foreground font-mono">
                                  {holding.ticker}
                                </div>
                              )}
                            </div>
                          </div>
                          <Badge variant="secondary" className="font-mono">
                            {holding.weight.toFixed(2)}%
                          </Badge>
                        </div>
                      ))}
                    </div>
                    
                    {hasMoreHoldings && (
                      <Button 
                        variant="ghost" 
                        size="sm" 
                        className="w-full"
                        onClick={() => setHoldingsExpanded(!holdingsExpanded)}
                      >
                        {holdingsExpanded ? (
                          <>
                            <ChevronUp className="h-4 w-4 mr-2" />
                            Show Less
                          </>
                        ) : (
                          <>
                            <ChevronDown className="h-4 w-4 mr-2" />
                            Show All ({topHoldings.length} holdings)
                          </>
                        )}
                      </Button>
                    )}
                  </div>

                  <Separator />
                </>
              )}

              {/* Asset Breakdown */}
              {etf.assetBreakdown && (
                <div className="space-y-3">
                  <h4 className="font-semibold flex items-center gap-2">
                    <PieChart className="h-4 w-4" />
                    Asset Breakdown
                  </h4>
                  <div className="grid grid-cols-2 gap-2 text-sm">
                    {etf.assetBreakdown.equities > 0 && (
                      <div className="flex justify-between p-2 border rounded bg-muted/10">
                        <span>Equities</span>
                        <span className="font-mono">{etf.assetBreakdown.equities}%</span>
                      </div>
                    )}
                    {etf.assetBreakdown.bonds > 0 && (
                      <div className="flex justify-between p-2 border rounded bg-muted/10">
                        <span>Bonds</span>
                        <span className="font-mono">{etf.assetBreakdown.bonds}%</span>
                      </div>
                    )}
                    {etf.assetBreakdown.cash > 0 && (
                      <div className="flex justify-between p-2 border rounded bg-muted/10">
                        <span>Cash</span>
                        <span className="font-mono">{etf.assetBreakdown.cash}%</span>
                      </div>
                    )}
                    {etf.assetBreakdown.commodities > 0 && (
                      <div className="flex justify-between p-2 border rounded bg-muted/10">
                        <span>Commodities</span>
                        <span className="font-mono">{etf.assetBreakdown.commodities}%</span>
                      </div>
                    )}
                    {etf.assetBreakdown.other > 0 && (
                      <div className="flex justify-between p-2 border rounded bg-muted/10">
                        <span>Other</span>
                        <span className="font-mono">{etf.assetBreakdown.other}%</span>
                      </div>
                    )}
                  </div>
                </div>
              )}

              <Separator />

              {/* Eligibility Justification */}
              <div className="space-y-3">
                <h4 className="font-semibold flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4" />
                  Eligibility Assessment
                </h4>
                <p className="text-sm bg-muted/30 p-3 rounded-lg">
                  {etf.eligibility.justification}
                </p>
                {etf.eligibility.ruleVersion && (
                  <div className="text-xs text-muted-foreground">
                    Rule Version: <code className="font-mono">{etf.eligibility.ruleVersion}</code>
                  </div>
                )}

                {/* Rules Passed */}
                {etf.eligibility.rulesPassed && etf.eligibility.rulesPassed.length > 0 && (
                  <div className="space-y-2">
                    <div className="text-sm font-medium text-emerald-700">Rules Passed</div>
                    <div className="flex flex-wrap gap-2">
                      {etf.eligibility.rulesPassed.map((rule, i) => (
                        <Badge key={i} variant="outline" className="bg-emerald-500/10 text-emerald-700 border-emerald-500/20">
                          <CheckCircle2 className="h-3 w-3 mr-1" />
                          {rule}
                        </Badge>
                      ))}
                    </div>
                  </div>
                )}

                {/* Rules Failed */}
                {etf.eligibility.rulesFailed && etf.eligibility.rulesFailed.length > 0 && (
                  <div className="space-y-2">
                    <div className="text-sm font-medium text-destructive">Rules Failed</div>
                    <div className="flex flex-wrap gap-2">
                      {etf.eligibility.rulesFailed.map((rule, i) => (
                        <Badge key={i} variant="outline" className="bg-destructive/10 text-destructive border-destructive/20">
                          <AlertTriangle className="h-3 w-3 mr-1" />
                          {rule}
                        </Badge>
                      ))}
                    </div>
                  </div>
                )}
              </div>

              <Separator />

              {/* Data Sources */}
              {etf.dataSources && etf.dataSources.length > 0 && (
                <div className="space-y-3">
                  <h4 className="font-semibold flex items-center gap-2">
                    <Database className="h-4 w-4" />
                    Data Sources
                  </h4>
                  <div className="space-y-2">
                    {etf.dataSources.map((source, i) => (
                      <a
                        key={i}
                        href={source.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="flex items-center justify-between p-3 border rounded-lg hover:bg-muted/30 transition-colors group"
                      >
                        <div>
                          <div className="font-medium">{source.provider}</div>
                          <div className="text-xs text-muted-foreground">
                            {source.type} • {source.date}
                          </div>
                        </div>
                        <ExternalLink className="h-4 w-4 text-muted-foreground group-hover:text-primary transition-colors" />
                      </a>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}
