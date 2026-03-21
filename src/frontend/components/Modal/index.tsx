import { useRef, useEffect, useState, KeyboardEvent } from 'react';
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
  const [isModalOpen, setModalOpen] = useState(isOpen);
  const modalRef = useRef<HTMLDialogElement>(null);

  const handleCloseModal = () => {
    if (onClose) onClose();
    setModalOpen(false);
  };

  const handleKeyDown = (event: KeyboardEvent) => {
    if (event.key === 'Escape') handleCloseModal();
  };

  useEffect(() => { setModalOpen(isOpen) }, [isOpen]);

  useEffect(() => {
    const modalElement = modalRef.current;
    if (isModalOpen) modalElement?.showModal?.();
    else modalElement?.close?.();
  }, [isModalOpen]);

  return (
    <dialog
      ref={modalRef}
      onKeyDown={handleKeyDown}
      onClick={(event) => {
        if (event.target === event.currentTarget) handleCloseModal();
      }}
      className={`modal ${size === 'compact' ? 'modal-compact' : ''}`.trim()}
      aria-modal="true"
    >
      {hasCloseBtn && (
        <button onClick={handleCloseModal}
          className="basic-button reverse-button modal-close-btn" >
          CLOSE
        </button>
      )}
      {children}
    </dialog>
  );
};

export default Modal;
