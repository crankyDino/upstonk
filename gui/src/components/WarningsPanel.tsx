import { useDiscoveryStore } from '@/stores/discoveryStore';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { AlertTriangle, Info, XCircle, RefreshCw } from 'lucide-react';

export function WarningsPanel() {
  const { response, error, isLoading, submitDiscovery } = useDiscoveryStore();

  if (error) {
    return (
      <Alert variant="destructive" className="mb-6">
        <XCircle className="h-4 w-4" />
        <AlertTitle>API Error</AlertTitle>
        <AlertDescription className="mt-2">
          <p>{error.message}</p>
          {error.requestId && (
            <p className="text-xs mt-2">
              Request ID: <code className="font-mono">{error.requestId}</code>
            </p>
          )}
          <Button 
            variant="outline" 
            size="sm" 
            className="mt-3"
            onClick={submitDiscovery}
            disabled={isLoading}
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
            Retry Request
          </Button>
        </AlertDescription>
      </Alert>
    );
  }

  if (!response || response.warnings.length === 0) return null;

  const severityIcons = {
    info: Info,
    warning: AlertTriangle,
    error: XCircle,
  };

  const severityStyles = {
    info: 'bg-blue-500/10 text-blue-700 border-blue-500/20',
    warning: 'bg-amber-500/10 text-amber-700 border-amber-500/20',
    error: 'bg-destructive/10 text-destructive border-destructive/20',
  };

  return (
    <div className="space-y-3 mb-6">
      {response.warnings.map((warning, i) => {
        const Icon = severityIcons[warning.severity];
        return (
          <div 
            key={i}
            className={`flex items-start gap-3 p-4 rounded-lg border ${severityStyles[warning.severity]}`}
          >
            <Icon className="h-5 w-5 shrink-0 mt-0.5" />
            <div>
              <div className="font-medium">{warning.code}</div>
              <div className="text-sm mt-0.5">{warning.message}</div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
