import { useState } from 'react';
import { useDiscoveryStore } from '@/stores/discoveryStore';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Button } from '@/components/ui/button';
import { EligibilityBadge } from './EligibilityBadge';
import { formatCurrency, formatPercentage } from '@/lib/api/upstonk';
import { ArrowUpDown, ArrowUp, ArrowDown, ChevronRight } from 'lucide-react';
import type { ETFResult } from '@/types/api';

type SortField = 'rank' | 'ticker' | 'ter' | 'aum' | 'matchScore' | 'rankingScore';
type SortDirection = 'asc' | 'desc';

export function ResultsTable() {
  const { response, setSelectedETF } = useDiscoveryStore();
  const [sortField, setSortField] = useState<SortField>('rank');
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc');

  if (!response) return null;

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDirection('asc');
    }
  };

  const sortedResults = [...response.results].sort((a, b) => {
    const modifier = sortDirection === 'asc' ? 1 : -1;
    const aValue = a[sortField];
    const bValue = b[sortField];
    
    if (typeof aValue === 'string' && typeof bValue === 'string') {
      return aValue.localeCompare(bValue) * modifier;
    }
    return ((aValue as number) - (bValue as number)) * modifier;
  });

  const SortIcon = ({ field }: { field: SortField }) => {
    if (sortField !== field) return <ArrowUpDown className="h-4 w-4" />;
    return sortDirection === 'asc' ? <ArrowUp className="h-4 w-4" /> : <ArrowDown className="h-4 w-4" />;
  };

  return (
    <div className="border rounded-lg overflow-hidden bg-card">
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow className="bg-muted/30">
              <TableHead className="w-16">
                <Button variant="ghost" size="sm" onClick={() => handleSort('rank')} className="h-8 px-2">
                  Rank <SortIcon field="rank" />
                </Button>
              </TableHead>
              <TableHead>
                <Button variant="ghost" size="sm" onClick={() => handleSort('ticker')} className="h-8 px-2">
                  Ticker <SortIcon field="ticker" />
                </Button>
              </TableHead>
              <TableHead className="min-w-[200px]">Name</TableHead>
              <TableHead>Provider</TableHead>
              <TableHead>
                <Button variant="ghost" size="sm" onClick={() => handleSort('ter')} className="h-8 px-2">
                  TER <SortIcon field="ter" />
                </Button>
              </TableHead>
              <TableHead>
                <Button variant="ghost" size="sm" onClick={() => handleSort('aum')} className="h-8 px-2">
                  AUM <SortIcon field="aum" />
                </Button>
              </TableHead>
              <TableHead>
                <Button variant="ghost" size="sm" onClick={() => handleSort('rankingScore')} className="h-8 px-2">
                  Score <SortIcon field="rankingScore" />
                </Button>
              </TableHead>
              <TableHead>Eligibility</TableHead>
              <TableHead className="w-10"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {sortedResults.map((etf) => (
              <TableRow 
                key={etf.ticker}
                className="cursor-pointer hover:bg-muted/50 transition-colors"
                onClick={() => setSelectedETF(etf.ticker)}
              >
                <TableCell className="font-medium text-center">{etf.rank}</TableCell>
                <TableCell>
                  <span className="font-mono font-semibold text-primary">{etf.ticker}</span>
                </TableCell>
                <TableCell className="font-medium">{etf.name || '-'}</TableCell>
                <TableCell className="text-muted-foreground">{etf.provider || '-'}</TableCell>
                <TableCell>{formatPercentage(etf.ter)}</TableCell>
                <TableCell>{formatCurrency(etf.aum)}</TableCell>
                <TableCell>
                  <span className="font-semibold">{etf.rankingScore.toFixed(1)}</span>
                </TableCell>
                <TableCell>
                  <EligibilityBadge 
                    status={etf.eligibility.status} 
                    confidence={etf.eligibility.confidence}
                    ruleVersion={etf.eligibility.ruleVersion}
                    size="sm"
                  />
                </TableCell>
                <TableCell>
                  <ChevronRight className="h-4 w-4 text-muted-foreground" />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      
      <div className="px-4 py-3 border-t bg-muted/20 flex items-center justify-between text-sm text-muted-foreground">
        <span>
          Showing {response.results.length} of {response.summary.totalSearched} searched ETFs
          {response.summary.totalEligible > 0 && (
            <span className="ml-2 text-emerald-600">• {response.summary.totalEligible} eligible</span>
          )}
        </span>
        <span>
          Request ID: <code className="font-mono text-xs">{response.requestId}</code>
        </span>
      </div>
    </div>
  );
}
