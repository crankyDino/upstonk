import { Badge } from '@/components/ui/badge';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import type { EligibilityStatus, ConfidenceLevel } from '@/types/api';
import { CheckCircle2, XCircle, HelpCircle, AlertCircle } from 'lucide-react';

interface EligibilityBadgeProps {
  status: EligibilityStatus;
  confidence: ConfidenceLevel;
  ruleVersion?: string;
  size?: 'sm' | 'default';
}

const statusConfig: Record<EligibilityStatus, { 
  label: string; 
  variant: 'default' | 'secondary' | 'destructive' | 'outline';
  className: string;
  icon: typeof CheckCircle2;
}> = {
  eligible: { 
    label: 'Eligible', 
    variant: 'default',
    className: 'bg-emerald-500/10 text-emerald-700 border-emerald-500/20 hover:bg-emerald-500/20',
    icon: CheckCircle2,
  },
  ineligible: { 
    label: 'Ineligible', 
    variant: 'destructive',
    className: 'bg-destructive/10 text-destructive border-destructive/20 hover:bg-destructive/20',
    icon: XCircle,
  },
  unknown: { 
    label: 'Unknown', 
    variant: 'secondary',
    className: 'bg-amber-500/10 text-amber-700 border-amber-500/20 hover:bg-amber-500/20',
    icon: HelpCircle,
  },
  conditional: { 
    label: 'Conditional', 
    variant: 'outline',
    className: 'bg-blue-500/10 text-blue-700 border-blue-500/20 hover:bg-blue-500/20',
    icon: AlertCircle,
  },
};

const confidenceLabels: Record<ConfidenceLevel, string> = {
  high: 'High confidence',
  medium: 'Medium confidence',
  low: 'Low confidence',
  unknown: 'Confidence unknown',
};

export function EligibilityBadge({ status, confidence, ruleVersion, size = 'default' }: EligibilityBadgeProps) {
  const config = statusConfig[status];
  const Icon = config.icon;
  
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Badge 
          variant={config.variant}
          className={`${config.className} ${size === 'sm' ? 'text-xs px-2 py-0.5' : 'px-3 py-1'} cursor-help font-medium`}
        >
          <Icon className={`${size === 'sm' ? 'h-3 w-3' : 'h-4 w-4'} mr-1`} />
          {config.label}
        </Badge>
      </TooltipTrigger>
      <TooltipContent className="max-w-xs">
        <div className="space-y-1">
          <p className="font-medium">{confidenceLabels[confidence]}</p>
          {ruleVersion && (
            <p className="text-xs text-muted-foreground">
              Rule version: {ruleVersion}
            </p>
          )}
        </div>
      </TooltipContent>
    </Tooltip>
  );
}
