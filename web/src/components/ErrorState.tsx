import { AlertTriangle, RefreshCw } from 'lucide-react';

interface ErrorStateProps {
  message?: string;
  onRetry?: () => void;
}

export function ErrorState({ message = 'Something went wrong', onRetry }: ErrorStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-center">
      <div className="p-3 rounded-full bg-[var(--color-error)]/10 mb-4">
        <AlertTriangle size={24} className="text-[var(--color-error)]" />
      </div>
      <p className="text-sm text-[var(--color-text-muted)] mb-4">{message}</p>
      {onRetry && (
        <button
          onClick={onRetry}
          className="flex items-center gap-2 text-sm text-[var(--color-primary)] hover:text-[var(--color-primary-hover)] transition-colors"
        >
          <RefreshCw size={14} /> Try again
        </button>
      )}
    </div>
  );
}
