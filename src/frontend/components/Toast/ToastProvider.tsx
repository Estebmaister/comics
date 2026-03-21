import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  useEffect,
} from 'react';
import type { ReactNode } from 'react';
import type { ToastInput } from '../Comics/types';
import styles from './ToastProvider.module.css';

type ToastTone = NonNullable<ToastInput['tone']>;

type ToastItem = Required<Pick<ToastInput, 'title'>> & {
  id: number;
  description?: string;
  tone: ToastTone;
  duration: number;
};

type ToastContextValue = {
  notify: (toast: ToastInput) => void;
  success: (toast: Omit<ToastInput, 'tone'>) => void;
  error: (toast: Omit<ToastInput, 'tone'>) => void;
  info: (toast: Omit<ToastInput, 'tone'>) => void;
  dismiss: (id: number) => void;
};

const SUCCESS_DURATION = 3500;
const ERROR_DURATION = 6000;
const INFO_DURATION = 4000;

const ToastContext = createContext<ToastContextValue | null>(null);

const toastDuration = (tone: ToastTone) => {
  switch (tone) {
    case 'error':
      return ERROR_DURATION;
    case 'info':
      return INFO_DURATION;
    default:
      return SUCCESS_DURATION;
  }
};

const ToastCard = ({
  toast,
  onDismiss,
}: {
  toast: ToastItem;
  onDismiss: (id: number) => void;
}) => {
  useEffect(() => {
    const timeoutId = window.setTimeout(() => {
      onDismiss(toast.id);
    }, toast.duration);

    return () => window.clearTimeout(timeoutId);
  }, [onDismiss, toast.duration, toast.id]);

  return (
    <article
      className={`${styles.toastCard} ${styles[`toast${toast.tone[0].toUpperCase()}${toast.tone.slice(1)}`]}`}
      role={toast.tone === 'error' ? 'alert' : 'status'}
      aria-live={toast.tone === 'error' ? 'assertive' : 'polite'}
    >
      <div className={styles.toastCopy}>
        <strong className={styles.toastTitle}>{toast.title}</strong>
        {toast.description ? (
          <p className={styles.toastDescription}>{toast.description}</p>
        ) : null}
      </div>

      <button
        type="button"
        className={styles.dismissButton}
        onClick={() => onDismiss(toast.id)}
        aria-label={`Dismiss notification: ${toast.title}`}
      >
        x
      </button>
    </article>
  );
};

export const ToastProvider = ({ children }: { children: ReactNode }) => {
  const [toasts, setToasts] = useState<ToastItem[]>([]);

  const dismiss = useCallback((id: number) => {
    setToasts((current) => current.filter((toast) => toast.id !== id));
  }, []);

  const notify = useCallback((toast: ToastInput) => {
    const tone = toast.tone ?? 'success';
    const nextToast: ToastItem = {
      id: Date.now() + Math.floor(Math.random() * 1000),
      title: toast.title,
      description: toast.description,
      tone,
      duration: toast.duration ?? toastDuration(tone),
    };

    setToasts((current) => [...current, nextToast]);
  }, []);

  const value = useMemo<ToastContextValue>(() => ({
    notify,
    dismiss,
    success: (toast) => notify({ ...toast, tone: 'success' }),
    error: (toast) => notify({ ...toast, tone: 'error' }),
    info: (toast) => notify({ ...toast, tone: 'info' }),
  }), [dismiss, notify]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <section
        className={styles.toastViewport}
        aria-label="Notifications"
        data-testid="toast-viewport"
      >
        {toasts.map((toast) => (
          <ToastCard key={toast.id} toast={toast} onDismiss={dismiss} />
        ))}
      </section>
    </ToastContext.Provider>
  );
};

export const useToast = () => {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within ToastProvider');
  }
  return context;
};
