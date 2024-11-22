import React, { useState, useEffect, useRef, ChangeEvent } from 'react';
import './CreateModal.css';
import Modal from '../../Modal';
import InputDiv from './InputDiv';
import PropTypes from 'prop-types';
import db_classes from '../../../../db/db_classes.json'

const createComicEmpty = {
  title: '',
  track: false,
  current_chap: 0,
  viewed_chap: 0,
  cover: 'https://',
  description: '',
  author: '',

  com_type: 3,
  status: 2,
  published_in: 0,
  genres: 0,
};

const formType = (field: string) => {
  switch (field) {
    case 'track':
      return 'checkbox';
    case 'viewed_chap':
    case 'current_chap':
      return 'number';
    case 'cover':
      return 'url';
    case 'com_type':
    case 'status':
    case 'published_in':
    case 'genres':
      return 'select';
    default:
      return 'text';
  }
}

const CreateComicModal = ({ onSubmit, isOpen, onClose }: any) => {
  const focusInputRef = useRef<any>(null);
  const [formState, setFormState] = useState(createComicEmpty);

  useEffect(() => {
    if (isOpen && focusInputRef.current) {
      setTimeout(() => { focusInputRef.current.focus(); }, 0);
    }
  }, [isOpen]);

  const handleInputChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { name, value, checked, type } = event.target;
    let newEntry: any;
    if (type === 'select-one') newEntry = parseInt(value);
    else if (type === 'checkbox') newEntry = checked;
    else newEntry = value;
    setFormState((prevFormData) => ({
      ...prevFormData,
      [name]: newEntry,
    }));
  };

  const handleSubmit = async (event: ChangeEvent<HTMLInputElement>) => {
    event.preventDefault();
    if (await onSubmit(formState)) setFormState(createComicEmpty);
  };

  return (
    <Modal hasCloseBtn={true} isOpen={isOpen} onClose={onClose}>
      <form onSubmit={() => handleSubmit}>

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
          />
        )}

        <div className='form-row'>
          <button className='basic-button' type='submit'>CREATE</button>
        </div>
      </form>
    </Modal>
  );
};

CreateComicModal.propTypes = {
  onSubmit: PropTypes.func.isRequired,
  isOpen: PropTypes.bool.isRequired,
  onClose: PropTypes.func.isRequired
};

export default CreateComicModal;