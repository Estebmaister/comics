import { useState, useEffect, useRef, FormEvent } from 'react';
import './Modal.css';
import Modal from '../../Modal';
import InputDiv from './InputDiv';
import db_classes from '../../../../db/db_classes.json'
import { handleInputChange, ComicModalProps } from './helpers';
import type { Comic } from '../types';

const formType = (field: string) => {
  switch (field) {
    case 'description':
      return 'textarea';
    case 'viewed_chap':
    case 'current_chap':
    case 'rating':
      return 'number';
    case 'cover':
      return 'url';
    case 'com_type':
    case 'status':
    case 'published_in':
    case 'genres':
      return 'select';
    case 'id':
    case 'last_update':
    case 'deleted':
    case 'cover_visible':
    case 'track':
      return 'none';
    default:
      return 'text';
  }
}

const EditComicModal: React.FC<ComicModalProps<Comic>> = ({ comic, isOpen, onSubmit, onClose }) => {
  const focusInputRef = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (isOpen && focusInputRef.current) {
      setTimeout(() => { focusInputRef.current?.focus(); }, 0);
    }
  }, [isOpen]);

  const [formState, setFormState] = useState<Comic>(comic as Comic);
  useEffect(() => setFormState(comic as Comic), [comic]);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (await onSubmit(formState)) setFormState((comic ?? formState) as Comic);
  };

  return (
    <Modal hasCloseBtn={true} isOpen={isOpen} onClose={onClose}>
      <form className='comic-modal-form' onSubmit={handleSubmit}>
        <header className='modal-form-header'>
          <h2>Edit Comic</h2>
          <p>Keep the data accurate without losing reading rhythm. Longer fields expand cleanly, and the action footer stays anchored while you scroll.</p>
        </header>

        <div className='form-grid'>
          {Object.entries(formState).map(([kField, value], _i) =>
            <InputDiv
              key={kField}
              value={value}
              field={kField}
              focusInputRef={focusInputRef}
              selectOptDict={db_classes}
              className={`form-row${kField === 'description' || kField === 'titles' ? ' form-row-span-2' : ''}`}
              type={formType(kField)}
              handleInputChange={handleInputChange(setFormState)}
              multiple={kField === 'genres' || kField === 'published_in'}
            />
          )}
        </div>

        <div className='form-actions'>
          <button className='basic-button' type='submit'>Update</button>
        </div>
      </form>
    </Modal>
  );
};

export default EditComicModal;
