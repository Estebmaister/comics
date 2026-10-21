import { useEffect, useRef } from 'react';

export const NAV_HEIGHT_UPDATED_EVENT = 'comics:nav-height-updated';

function publishNavHeight(height: number) {
  const root = document.documentElement;
  const value = `${Math.ceil(height)}px`;
  if (root.style.getPropertyValue('--nav-height') === value) {
    return;
  }
  root.style.setProperty('--nav-height', value);
  window.dispatchEvent(new CustomEvent(NAV_HEIGHT_UPDATED_EVENT, { detail: height }));
}

/**
 * Keeps --nav-height in sync with the fixed header so list padding clears the toolbar.
 */
export function useMeasuredNavHeight<T extends HTMLElement>() {
  const ref = useRef<T | null>(null);

  useEffect(() => {
    const element = ref.current;
    if (!element) return undefined;

    const measure = () => {
      publishNavHeight(element.getBoundingClientRect().height);
    };

    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(element);
    window.addEventListener('resize', measure);

    return () => {
      observer.disconnect();
      window.removeEventListener('resize', measure);
    };
  }, []);

  return ref;
}
