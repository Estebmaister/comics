import { useState, useEffect, useRef, FormEvent } from 'react';
import './Modal.css';
import Modal from '../../Modal';
import InputDiv from './InputDiv';
import db_classes from '../../../../db/db_classes.json'
import { handleInputChange, ComicModalProps } from './helpers';

const formType = (field: string) => {
  switch (field) {
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
    case 'track':
      return 'none';
    default:
      return 'text';
  }
}

const EditComicModal: React.FC<ComicModalProps> = ({ comic, isOpen, onSubmit, onClose }) => {
  const focusInputRef = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (isOpen && focusInputRef.current) {
      setTimeout(() => { focusInputRef.current?.focus(); }, 0);
    }
  }, [isOpen]);

  const [formState, setFormState] = useState(comic || {});
  useEffect(() => setFormState(comic || {}), [comic]);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (await onSubmit(formState)) setFormState(comic || {});
  };

  return (
    <Modal hasCloseBtn={true} isOpen={isOpen} onClose={onClose}>
      <form onSubmit={handleSubmit}>

        {Object.entries(formState).map(([kField, value], _i) =>
          <InputDiv
            key={kField}
            value={value}
            field={kField}
            focusInputRef={focusInputRef}
            selectOptDict={db_classes}
            className={'form-row'}
            type={formType(kField)}
            handleInputChange={handleInputChange(setFormState)}
            multiple={kField === 'genres' || kField === 'published_in'}
          />
        )}

        <div className='form-row'>
          <button className='basic-button' type='submit'>UPDATE</button>
        </div>
      </form>
    </Modal>
  );
};

export default EditComicModal;