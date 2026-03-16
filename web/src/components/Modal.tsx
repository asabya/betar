import { type ReactNode, useEffect, useRef, useId } from 'react';
import { X } from 'lucide-react';

interface ModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
}

export function Modal({ open, onClose, title, children }: ModalProps) {
  const dialogRef = useRef<HTMLDivElement>(null);
  const previousFocusRef = useRef<HTMLElement | null>(null);
  const titleId = useId();

  useEffect(() => {
    if (!open) return;

    // Save previously focused element
    previousFocusRef.current = document.activeElement as HTMLElement;

    // Focus the dialog
    dialogRef.current?.focus();

    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
        return;
      }

      // Focus trap
      if (e.key === 'Tab' && dialogRef.current) {
        const focusable = dialogRef.current.querySelectorAll<HTMLElement>(
          'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        );
        const first = focusable[0];
        const last = focusable[focusable.length - 1];

        if (e.shiftKey) {
          if (document.activeElement === first) {
            e.preventDefault();
            last?.focus();
          }
        } else {
          if (document.activeElement === last) {
            e.preventDefault();
            first?.focus();
          }
        }
      }
    };
    window.addEventListener('keydown', handler);

    return () => {
      window.removeEventListener('keydown', handler);
      // Restore focus on close
      previousFocusRef.current?.focus();
    };
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-[var(--color-overlay)] backdrop-blur-sm" aria-hidden="true" onClick={onClose} />
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        className="relative bg-[var(--color-surface-solid)] border border-[var(--color-border-hover)] rounded-2xl w-full max-w-lg max-h-[90vh] overflow-auto p-6 shadow-2xl outline-none"
      >
        <div className="flex items-center justify-between mb-5">
          <h2 id={titleId} className="text-lg font-semibold" style={{ fontFamily: 'var(--font-display)' }}>{title}</h2>
          <button
            onClick={onClose}
            aria-label="Close dialog"
            className="text-[var(--color-text-dim)] hover:text-[var(--color-text)] transition-colors p-1 rounded-lg hover:bg-[var(--color-primary-subtle)]"
          >
            <X size={18} />
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}
