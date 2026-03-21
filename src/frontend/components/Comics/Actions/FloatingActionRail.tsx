import type { ButtonHTMLAttributes, ReactNode } from 'react';
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

export const FloatingActionRail = ({ children }: FloatingActionRailProps) => (
  <aside className={styles.railShell} aria-label="Comic utilities">
    <div className={styles.rail}>{children}</div>
  </aside>
);

export const RailActionButton = ({
  eyebrow,
  title,
  description,
  tone = 'cool',
  className = '',
  ...props
}: RailActionButtonProps) => (
  <button
    type="button"
    className={`${styles.actionButton} ${styles[`tone${tone[0].toUpperCase()}${tone.slice(1)}`]} ${className}`.trim()}
    {...props}
  >
    <span className={styles.actionEyebrow}>{eyebrow}</span>
    <span className={styles.actionTitle}>{title}</span>
    <span className={styles.actionDescription}>{description}</span>
  </button>
);
