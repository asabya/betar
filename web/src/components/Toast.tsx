import { createContext, useContext, useState, useCallback, useEffect } from 'react';
import { CheckCircle2, XCircle, Info, X } from 'lucide-react';

type ToastVariant = 'success' | 'error' | 'info';

interface Toast {
  id: number;
  message: string;
  variant: ToastVariant;
}

interface ToastContextValue {
  toast: (message: string, variant?: ToastVariant) => void;
}

const ToastContext = createContext<ToastContextValue>({ toast: () => {} });

export function useToast() {
  return useContext(ToastContext);
}

let nextId = 0;

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const toast = useCallback((message: string, variant: ToastVariant = 'info') => {
    const id = nextId++;
    setToasts((prev) => [...prev, { id, message, variant }]);
  }, []);

  const dismiss = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  return (
    <ToastContext.Provider value={{ toast }}>
      {children}
      <div
        className="fixed bottom-20 lg:bottom-4 right-4 z-[60] flex flex-col gap-2 max-w-sm"
        aria-live="polite"
        aria-atomic="false"
      >
        {toasts.map((t) => (
          <ToastItem key={t.id} toast={t} onDismiss={() => dismiss(t.id)} />
        ))}
      </div>
    </ToastContext.Provider>
  );
}

const icons: Record<ToastVariant, React.ElementType> = {
  success: CheckCircle2,
  error: XCircle,
  info: Info,
};

const colors: Record<ToastVariant, string> = {
  success: 'border-[var(--color-success)]/20 bg-[var(--color-success)]/10 text-[var(--color-success)]',
  error: 'border-[var(--color-error)]/20 bg-[var(--color-error)]/10 text-[var(--color-error)]',
  info: 'border-[var(--color-border-hover)] bg-[var(--color-primary-subtle)] text-[var(--color-primary-light)]',
};

const roles: Record<ToastVariant, string> = {
  success: 'status',
  error: 'alert',
  info: 'status',
};

function ToastItem({ toast, onDismiss }: { toast: Toast; onDismiss: () => void }) {
  const Icon = icons[toast.variant];

  useEffect(() => {
    const timer = setTimeout(onDismiss, 3000);
    return () => clearTimeout(timer);
  }, [onDismiss]);

  return (
    <div
      role={roles[toast.variant]}
      className={`flex items-start gap-3 px-4 py-3 rounded-xl border backdrop-blur-xl shadow-lg ${colors[toast.variant]}`}
    >
      <Icon size={18} className="shrink-0 mt-0.5" aria-hidden="true" />
      <p className="text-sm flex-1">{toast.message}</p>
      <button
        onClick={onDismiss}
        aria-label="Dismiss notification"
        className="shrink-0 opacity-60 hover:opacity-100 transition-opacity"
      >
        <X size={14} />
      </button>
    </div>
  );
}
