import { useCallback, useEffect, useLayoutEffect, useRef } from 'react';
import { createPortal } from 'react-dom';
import './Modal.css';

interface ModalProps {
  isOpen: boolean;
  hasCloseBtn?: boolean;
  size?: 'default' | 'compact';
  onClose?: () => void;
  children: React.ReactNode;
}

const Modal = ({
  isOpen,
  hasCloseBtn = true,
  size = 'default',
  onClose,
  children,
}: ModalProps) => {
  const previouslyFocusedRef = useRef<HTMLElement | null>(null);

  const handleCloseModal = useCallback(() => {
    if (onClose) onClose();
  }, [onClose]);

  useLayoutEffect(() => {
    if (!isOpen) return undefined;

    previouslyFocusedRef.current = document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null;

    return () => {
      previouslyFocusedRef.current?.focus();
      previouslyFocusedRef.current = null;
    };
  }, [isOpen]);

  useEffect(() => {
    if (!isOpen) return undefined;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') handleCloseModal();
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleCloseModal, isOpen]);

  if (!isOpen) return null;

  return createPortal(
    <div
      className="modal-backdrop"
      onClick={(event) => {
        if (event.target === event.currentTarget) handleCloseModal();
      }}
    >
      <div
        className={`modal ${size === 'compact' ? 'modal-compact' : ''}`.trim()}
        role="dialog"
        aria-modal="true"
      >
        {hasCloseBtn && (
          <button onClick={handleCloseModal}
            className="basic-button reverse-button modal-close-btn" >
            CLOSE
          </button>
        )}
        {children}
      </div>
    </div>,
    document.body,
  );
};

export default Modal;
