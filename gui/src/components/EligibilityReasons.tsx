import { AlertTriangle, CheckCircle2, Info, XCircle } from 'lucide-react';

// The API sends the assessment as one string: reasons joined by "; ", each
// prefixed ✓ (passed), ✗ (failed) or ⚠ (needs checking). Show it as a list.
const kinds = {
  '✓': { Icon: CheckCircle2, className: 'text-emerald-600' },
  '✗': { Icon: XCircle, className: 'text-destructive' },
  '⚠': { Icon: AlertTriangle, className: 'text-amber-600' },
} as const;

type Mark = keyof typeof kinds;

function parse(justification: string) {
  return justification
    .split(';')
    .map((s) => s.trim())
    .filter(Boolean)
    .map((s) => {
      const mark = s[0] as Mark;
      return mark in kinds ? { mark, text: s.slice(1).trim() } : { mark: null, text: s };
    });
}

export function EligibilityReasons({ justification }: { justification: string }) {
  const reasons = parse(justification);
  if (reasons.length === 0) return null;
  return (
    <ul className="space-y-2 text-sm">
      {reasons.map(({ mark, text }, i) => {
        const { Icon, className } = mark ? kinds[mark] : { Icon: Info, className: 'text-muted-foreground' };
        return (
          <li key={i} className="flex items-start gap-2">
            <Icon className={`h-4 w-4 shrink-0 mt-0.5 ${className}`} aria-hidden />
            <span className="min-w-0 break-words">{text}</span>
          </li>
        );
      })}
    </ul>
  );
}
