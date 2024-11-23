import { useRef, useEffect, useState, KeyboardEvent } from 'react';
import './Modal.css';

const Modal = ({ isOpen, hasCloseBtn = true, onClose, children }: any) => {
  const [isModalOpen, setModalOpen] = useState(isOpen);
  const modalRef = useRef<any>(null);

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

    if (modalElement) {
      if (isModalOpen) modalElement.showModal();
      else modalElement.close();
    }
  }, [isModalOpen]);

  return (
    <dialog ref={modalRef} onKeyDown={handleKeyDown} className="modal">
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
