import { useDiscoveryStore } from '@/stores/discoveryStore';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { User, MapPin, Wallet, Shield, Clock } from 'lucide-react';

const countries = [
  { code: 'CA', name: 'Canada' },
  { code: 'US', name: 'United States' },
  { code: 'GB', name: 'United Kingdom' },
  { code: 'AU', name: 'Australia' },
  { code: 'DE', name: 'Germany' },
  { code: 'FR', name: 'France' },
  { code: 'ZA', name: 'South Africa' },
];

const accountTypes = [
  { value: 'TFSA', label: 'TFSA (Canada)', description: 'Tax-Free Savings Account' },
  { value: 'ISA', label: 'ISA (UK)', description: 'Individual Savings Account' },
  { value: 'IRA', label: 'IRA (US)', description: 'Individual Retirement Account' },
  { value: 'standard', label: 'Standard', description: 'Taxable brokerage account' },
  { value: 'retirement_annuity', label: 'Retirement Annuity (ZA)', description: 'South African retirement fund' },
];

const currencies = [
  { code: 'CAD', name: 'Canadian Dollar' },
  { code: 'USD', name: 'US Dollar' },
  { code: 'GBP', name: 'British Pound' },
  { code: 'EUR', name: 'Euro' },
  { code: 'AUD', name: 'Australian Dollar' },
  { code: 'ZAR', name: 'South African Rand' },
];

const riskTolerances = [
  { value: 'conservative', label: 'Conservative', description: 'Lower risk, stable returns' },
  { value: 'moderate', label: 'Moderate', description: 'Balanced risk/return' },
  { value: 'aggressive', label: 'Aggressive', description: 'Higher risk, growth-focused' },
];

export function InvestorProfileForm() {
  const { investorProfile, setInvestorProfile } = useDiscoveryStore();

  return (
    <Card className="border-border/50 shadow-sm">
      <CardHeader className="pb-4">
        <CardTitle className="flex items-center gap-2 text-lg">
          <User className="h-5 w-5 text-primary" />
          Investor Profile
        </CardTitle>
        <CardDescription>
          Configure your investor details for accurate eligibility assessment
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="grid gap-6 md:grid-cols-2">
          {/* Country */}
          <div className="space-y-2">
            <Label className="flex items-center gap-2">
              <MapPin className="h-4 w-4 text-muted-foreground" />
              Country of Residence
            </Label>
            <Select
              value={investorProfile.country}
              onValueChange={(value) => setInvestorProfile({ country: value })}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select country" />
              </SelectTrigger>
              <SelectContent>
                {countries.map((country) => (
                  <SelectItem key={country.code} value={country.code}>
                    {country.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Account Type */}
          <div className="space-y-2">
            <Label className="flex items-center gap-2">
              <Wallet className="h-4 w-4 text-muted-foreground" />
              Account Type
            </Label>
            <Select
              value={investorProfile.accountType}
              onValueChange={(value) => setInvestorProfile({ accountType: value as any })}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select account type" />
              </SelectTrigger>
              <SelectContent>
                {accountTypes.map((type) => (
                  <SelectItem key={type.value} value={type.value}>
                    <div className="flex flex-col">
                      <span>{type.label}</span>
                      <span className="text-xs text-muted-foreground">{type.description}</span>
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Currency */}
          <div className="space-y-2">
            <Label>Preferred Currency</Label>
            <Select
              value={investorProfile.currency}
              onValueChange={(value) => setInvestorProfile({ currency: value })}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select currency" />
              </SelectTrigger>
              <SelectContent>
                {currencies.map((currency) => (
                  <SelectItem key={currency.code} value={currency.code}>
                    {currency.code} - {currency.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Risk Tolerance */}
          <div className="space-y-2">
            <Label className="flex items-center gap-2">
              <Shield className="h-4 w-4 text-muted-foreground" />
              Risk Tolerance
            </Label>
            <Select
              value={investorProfile.riskTolerance}
              onValueChange={(value) => setInvestorProfile({ riskTolerance: value as any })}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select risk tolerance" />
              </SelectTrigger>
              <SelectContent>
                {riskTolerances.map((risk) => (
                  <SelectItem key={risk.value} value={risk.value}>
                    <div className="flex flex-col">
                      <span>{risk.label}</span>
                      <span className="text-xs text-muted-foreground">{risk.description}</span>
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        {/* Time Horizon */}
        <div className="space-y-4">
          <Label className="flex items-center gap-2">
            <Clock className="h-4 w-4 text-muted-foreground" />
            Investment Time Horizon: {investorProfile.timeHorizon} years
          </Label>
          <Slider
            value={[investorProfile.timeHorizon]}
            onValueChange={([value]) => setInvestorProfile({ timeHorizon: value })}
            min={1}
            max={30}
            step={1}
            className="w-full"
          />
          <div className="flex justify-between text-xs text-muted-foreground">
            <span>1 year</span>
            <span>15 years</span>
            <span>30 years</span>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
