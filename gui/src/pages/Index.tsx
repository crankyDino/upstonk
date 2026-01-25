import { useState } from 'react';
import { useDiscoveryStore } from '@/stores/discoveryStore';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { InvestorProfileForm } from '@/components/InvestorProfileForm';
import { ExposureSelector } from '@/components/ExposureSelector';
import { ConstraintsPanel } from '@/components/ConstraintsPanel';
import { ResultsTable } from '@/components/ResultsTable';
import { ETFDetailDrawer } from '@/components/ETFDetailDrawer';
import { WarningsPanel } from '@/components/WarningsPanel';
import { 
  Search, 
  TrendingUp, 
  Shield, 
  Zap, 
  RotateCcw,
  Loader2,
  BarChart3,
  Clock
} from 'lucide-react';

const Index = () => {
  const { 
    isLoading, 
    response, 
    submitDiscovery, 
    resetForm,
    clearResults 
  } = useDiscoveryStore();

  const [activeTab, setActiveTab] = useState('profile');

  const handleSubmit = async () => {
    await submitDiscovery();
  };

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b bg-card/50 backdrop-blur-sm sticky top-0 z-50">
        <div className="container mx-auto px-4 h-16 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-primary flex items-center justify-center">
              <TrendingUp className="h-5 w-5 text-primary-foreground" />
            </div>
            <div>
              <h1 className="font-bold text-xl tracking-tight">UPSTONK</h1>
              <p className="text-xs text-muted-foreground">Intelligent ETF Discovery</p>
            </div>
          </div>
          
          <div className="flex items-center gap-3">
            <Button variant="ghost" size="sm" onClick={resetForm}>
              <RotateCcw className="h-4 w-4 mr-2" />
              Reset
            </Button>
          </div>
        </div>
      </header>

      <main className="container mx-auto px-4 py-8">
        {/* Hero Section */}
        {!response && (
          <div className="text-center mb-12">
            <h2 className="text-3xl font-bold mb-4">
              Discover Compliant ETF Investments
            </h2>
            <p className="text-muted-foreground max-w-2xl mx-auto">
              Find ETFs that match your investment goals with transparent eligibility 
              rules, explainable rankings, and source attribution you can trust.
            </p>
            
            <div className="flex items-center justify-center gap-8 mt-8">
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Shield className="h-4 w-4 text-emerald-600" />
                <span>Compliance-Aware</span>
              </div>
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Zap className="h-4 w-4 text-amber-600" />
                <span>Explainable Rankings</span>
              </div>
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <BarChart3 className="h-4 w-4 text-blue-600" />
                <span>Source Attribution</span>
              </div>
            </div>
          </div>
        )}

        {/* Discovery Form */}
        {!response && (
          <Card className="mb-8 border-border/50 shadow-lg">
            <CardHeader className="pb-4 border-b">
              <CardTitle className="flex items-center gap-2">
                <Search className="h-5 w-5" />
                ETF Discovery Query Builder
              </CardTitle>
              <CardDescription>
                Configure your search parameters to find matching ETFs
              </CardDescription>
            </CardHeader>
            <CardContent className="pt-6">
              <Tabs value={activeTab} onValueChange={setActiveTab}>
                <TabsList className="grid w-full grid-cols-3 mb-6">
                  <TabsTrigger value="profile">Investor Profile</TabsTrigger>
                  <TabsTrigger value="exposure">Exposure</TabsTrigger>
                  <TabsTrigger value="constraints">Constraints</TabsTrigger>
                </TabsList>
                
                <TabsContent value="profile" className="mt-0">
                  <InvestorProfileForm />
                </TabsContent>
                
                <TabsContent value="exposure" className="mt-0">
                  <ExposureSelector />
                </TabsContent>
                
                <TabsContent value="constraints" className="mt-0">
                  <ConstraintsPanel />
                </TabsContent>
              </Tabs>

              <div className="flex items-center justify-between mt-8 pt-6 border-t">
                <div className="flex gap-2">
                  {activeTab !== 'profile' && (
                    <Button 
                      variant="outline" 
                      onClick={() => setActiveTab(activeTab === 'constraints' ? 'exposure' : 'profile')}
                    >
                      Previous
                    </Button>
                  )}
                  {activeTab !== 'constraints' && (
                    <Button 
                      variant="outline"
                      onClick={() => setActiveTab(activeTab === 'profile' ? 'exposure' : 'constraints')}
                    >
                      Next
                    </Button>
                  )}
                </div>
                
                <Button 
                  size="lg" 
                  onClick={handleSubmit}
                  disabled={isLoading}
                  className="min-w-[200px]"
                >
                  {isLoading ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      Searching...
                    </>
                  ) : (
                    <>
                      <Search className="h-4 w-4 mr-2" />
                      Discover ETFs
                    </>
                  )}
                </Button>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Results Section */}
        {response && (
          <div className="space-y-6">
            {/* Results Header */}
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-2xl font-bold">Discovery Results</h2>
                <p className="text-muted-foreground flex items-center gap-2 mt-1">
                  <Clock className="h-4 w-4" />
                  {response.summary.searchDurationMs > 0 
                    ? `Query completed in ${(response.summary.searchDurationMs / 1000).toFixed(2)}s`
                    : `Generated at ${new Date(response.generatedAt).toLocaleTimeString()}`
                  }
                  {' • '}
                  {response.summary.dataSourcesQueried.length} sources queried
                </p>
              </div>
              <Button variant="outline" onClick={clearResults}>
                <Search className="h-4 w-4 mr-2" />
                New Search
              </Button>
            </div>
            
            {/* Summary Cards */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div className="p-4 border rounded-lg bg-card">
                <div className="text-2xl font-bold">{response.summary.totalSearched}</div>
                <div className="text-sm text-muted-foreground">Total Searched</div>
              </div>
              <div className="p-4 border rounded-lg bg-card">
                <div className="text-2xl font-bold text-emerald-600">{response.summary.totalEligible}</div>
                <div className="text-sm text-muted-foreground">Eligible</div>
              </div>
              <div className="p-4 border rounded-lg bg-card">
                <div className="text-2xl font-bold text-destructive">{response.summary.totalIneligible}</div>
                <div className="text-sm text-muted-foreground">Ineligible</div>
              </div>
              <div className="p-4 border rounded-lg bg-card">
                <div className="text-2xl font-bold text-amber-600">{response.summary.totalUnknown}</div>
                <div className="text-sm text-muted-foreground">Unknown</div>
              </div>
            </div>

            {/* Warnings */}
            <WarningsPanel />

            {/* Results Table */}
            <ResultsTable />

            {/* ETF Detail Drawer */}
            <ETFDetailDrawer />
          </div>
        )}
      </main>

      {/* Footer */}
      <footer className="border-t mt-16 py-8">
        <div className="container mx-auto px-4 text-center text-sm text-muted-foreground">
          <p>
            UPSTONK provides investment research tools only. 
            Always consult a financial advisor before making investment decisions.
          </p>
          <p className="mt-2">
            Data sources: Vanguard, iShares, SEC EDGAR, Morningstar • 
            Eligibility rules are versioned and auditable
          </p>
        </div>
      </footer>
    </div>
  );
};

export default Index;
