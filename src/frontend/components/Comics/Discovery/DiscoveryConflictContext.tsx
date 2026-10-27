import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react';
import type { CatalogConflictPayload } from '../../../util/catalogConflict';
import DiscoveryConflictModal from './DiscoveryConflictModal';
import { useToast } from '../../Toast/ToastProvider';

type PendingRetry = () => Promise<boolean>;

type OpenConflictArgs = {
  conflict: CatalogConflictPayload;
  retry?: PendingRetry;
  /** When false, skip the info toast (modal only). Default true. */
  notify?: boolean;
  onDismiss?: () => void;
};

type DiscoveryConflictContextValue = {
  openCatalogConflict: (args: OpenConflictArgs) => void;
};

const DiscoveryConflictContext = createContext<DiscoveryConflictContextValue | null>(null);

type DiscoveryConflictProviderProps = {
  children: ReactNode;
  onCatalogRefresh?: (removedComicIds: number[]) => void;
};

export const DiscoveryConflictProvider = ({
  children,
  onCatalogRefresh,
}: DiscoveryConflictProviderProps) => {
  const toast = useToast();
  const [isOpen, setIsOpen] = useState(false);
  const [conflict, setConflict] = useState<CatalogConflictPayload | null>(null);
  const retryRef = useRef<PendingRetry | undefined>(undefined);
  const dismissRef = useRef<(() => void) | undefined>(undefined);
  const [hasRetry, setHasRetry] = useState(false);

  const close = useCallback((dismiss = false) => {
    if (dismiss) {
      dismissRef.current?.();
    }
    setIsOpen(false);
    setConflict(null);
    retryRef.current = undefined;
    dismissRef.current = undefined;
    setHasRetry(false);
  }, []);

  const openCatalogConflict = useCallback((args: OpenConflictArgs) => {
    setConflict(args.conflict);
    retryRef.current = args.retry;
    dismissRef.current = args.onDismiss;
    setHasRetry(Boolean(args.retry));
    setIsOpen(true);
    if (args.notify !== false) {
      toast.info({
        title: 'Possible duplicate',
        description: 'Review these catalog entries and merge if they are the same series.',
      });
    }
  }, [toast]);

  const handleMerged = useCallback((removedComicId: number | null) => {
    onCatalogRefresh?.(removedComicId ? [removedComicId] : []);
    toast.success({
      title: 'Comics merged',
      description: 'Duplicate records were combined.',
    });
  }, [onCatalogRefresh, toast]);

  const handleRetry = useCallback(async () => {
    const retry = retryRef.current;
    if (!retry) return false;
    const ok = await retry();
    if (!ok) {
      toast.error({
        title: 'Action still blocked',
        description: 'Merge completed, but the original change could not be applied.',
      });
    }
    return ok;
  }, [toast]);

  const value = useMemo(
    () => ({ openCatalogConflict }),
    [openCatalogConflict],
  );

  return (
    <DiscoveryConflictContext.Provider value={value}>
      {children}
      <DiscoveryConflictModal
        isOpen={isOpen}
        conflict={conflict}
        onClose={() => close(false)}
        onDismiss={() => close(true)}
        onMerged={handleMerged}
        onRetry={hasRetry ? handleRetry : undefined}
      />
    </DiscoveryConflictContext.Provider>
  );
};

export const useDiscoveryConflict = (): DiscoveryConflictContextValue | null => (
  useContext(DiscoveryConflictContext)
);
