import React, { useState, useEffect, useRef } from 'react';
import './CreateModal.css';
import Modal from '../../Modal';
import InputDiv from './InputDiv';
import db_classes from '../../../../db/db_classes.json'

const createComicEmpty = {
  title: '',
  track: false,
  current_chap: 0,
  viewed_chap: 0,
  cover: '',
  description: '',
  author: '',

  published_in: 0,
  com_type: 0,
  status: 0,
  genres: 0,
};

const formType = (field) => {
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

const CreateComicModal = ({ onSubmit, isOpen, onClose }) => {
  const focusInputRef = useRef(null);
  const [formState, setFormState] = useState(createComicEmpty);

  useEffect(() => {
    if (isOpen && focusInputRef.current) {
      setTimeout(() => { focusInputRef.current.focus(); }, 0);
    }
  }, [isOpen]);

  const handleInputChange = (event) => {
    const { name, value } = event.target;
    setFormState((prevFormData) => ({
      ...prevFormData,
      [name]: value,
    }));
  };

  const handleSubmit = (event) => {
    event.preventDefault();
    onSubmit(formState);
    setFormState(createComicEmpty);
  };

  return (
    <Modal hasCloseBtn={true} isOpen={isOpen} onClose={onClose}>
      <form onSubmit={handleSubmit}>

        {Object.entries(formState).map( ([kField, value], _i) =>
          <InputDiv 
            key={kField} 
            value={value} 
            field={kField} 
            ref={focusInputRef}
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

export default CreateComicModal;