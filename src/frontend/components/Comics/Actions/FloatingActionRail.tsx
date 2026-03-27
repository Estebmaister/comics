import {
  createContext,
  useContext,
  useEffect,
  useId,
  useRef,
  useState,
} from 'react';
import type { ButtonHTMLAttributes, MouseEvent, ReactNode } from 'react';
import styles from './FloatingActionRail.module.css';

type FloatingActionRailProps = {
  children: ReactNode;
};

type RailActionButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  eyebrow: string;
  title: string;
  description: string;
  tone?: 'cool' | 'warm' | 'neutral';
};

type RailContextValue = {
  collapse: () => void;
};

const RailContext = createContext<RailContextValue | null>(null);

export const FloatingActionRail = ({ children }: FloatingActionRailProps) => {
  const [isOpen, setIsOpen] = useState(false);
  const panelId = useId();
  const shellRef = useRef<HTMLElement | null>(null);
  const launcherRef = useRef<HTMLButtonElement | null>(null);

  const collapseRail = () => {
    launcherRef.current?.focus();
    setIsOpen(false);
  };

  useEffect(() => {
    if (!isOpen) return;

    const handlePointerDown = (event: PointerEvent) => {
      if (!shellRef.current?.contains(event.target as Node)) {
        collapseRail();
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        collapseRail();
      }
    };

    document.addEventListener('pointerdown', handlePointerDown);
    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.removeEventListener('pointerdown', handlePointerDown);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [isOpen]);

  return (
    <RailContext.Provider value={{ collapse: collapseRail }}>
      <aside
        ref={shellRef}
        className={`${styles.railShell}${isOpen ? ` ${styles.railShellOpen}` : ''}`}
        aria-label="Comic utilities"
      >
        <div
          id={panelId}
          className={`${styles.railPanel}${isOpen ? ` ${styles.railPanelOpen}` : ''}`}
        >
          <div className={styles.rail}>{children}</div>
        </div>

        <button
          ref={launcherRef}
          type="button"
          className={`${styles.launcher}${isOpen ? ` ${styles.launcherOpen}` : ''}`}
          aria-controls={panelId}
          aria-expanded={isOpen}
          aria-label={isOpen ? 'Close comic utilities' : 'Open comic utilities'}
          onClick={() => setIsOpen((current) => !current)}
        >
          <span className={styles.launcherIcon} aria-hidden="true">
            <span className={styles.launcherLine} />
            <span className={styles.launcherLine} />
            <span className={styles.launcherLine} />
          </span>
        </button>
      </aside>
    </RailContext.Provider>
  );
};

export const RailActionButton = ({
  eyebrow,
  title,
  description,
  tone = 'cool',
  className = '',
  onClick,
  ...props
}: RailActionButtonProps) => {
  const rail = useContext(RailContext);

  const handleClick = (event: MouseEvent<HTMLButtonElement>) => {
    onClick?.(event);
    if (!event.defaultPrevented) {
      rail?.collapse();
    }
  };

  return (
    <button
      type="button"
      className={`${styles.actionButton} ${styles[`tone${tone[0].toUpperCase()}${tone.slice(1)}`]} ${className}`.trim()}
      onClick={handleClick}
      {...props}
    >
      <span className={styles.actionEyebrow}>{eyebrow}</span>
      <span className={styles.actionTitle}>{title}</span>
      <span className={styles.actionDescription}>{description}</span>
    </button>
  );
};
