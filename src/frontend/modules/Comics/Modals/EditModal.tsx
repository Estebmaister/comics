import { useState, useEffect, useRef, ChangeEvent, FormEvent } from 'react';
import './CreateModal.css';
import Modal from '../../Modal';
import InputDiv from './InputDiv';
import PropTypes from 'prop-types';
import db_classes from '../../../../db/db_classes.json'

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

const EditComicModal = ({ onSubmit, isOpen, onClose, comic }: any) => {
  const focusInputRef = useRef<any>(null);
  const [formState, setFormState] = useState(comic);

  useEffect(() => {
    if (isOpen && focusInputRef.current) {
      setTimeout(() => { focusInputRef.current.focus(); }, 0);
    }
  }, [isOpen]);

  const handleInputChange = (
    event: ChangeEvent<HTMLSelectElement> | ChangeEvent<HTMLInputElement> | any
  ) => {
    const { name, value, selectedOptions, checked, type } = event.target;
    let newEntry: any;
    if (type === 'select-one') newEntry = parseInt(value);
    else if (type === 'checkbox') newEntry = checked;
    else if (type === 'select-multiple') newEntry = Object
      .values(selectedOptions)?.map((options: any) => +options.value);
    else newEntry = value;
    setFormState((prevFormData: any) => ({
      ...prevFormData,
      [name]: newEntry,
    }));
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (await onSubmit(formState)) setFormState(comic);
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
            handleInputChange={handleInputChange}
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

EditComicModal.propTypes = {
  onSubmit: PropTypes.func.isRequired,
  isOpen: PropTypes.bool.isRequired,
  onClose: PropTypes.func.isRequired,
  comic: PropTypes.object.isRequired
};

export default EditComicModal;