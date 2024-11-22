import { useState, useEffect, useRef, ChangeEvent } from 'react';
import './CreateModal.css';
import Modal from '../../Modal';
import InputDiv from './InputDiv';
import PropTypes from 'prop-types';

const mergeEmptyDict = {
  baseID: 0,
  mergingID: 0,
};

const MergeComicModal = ({ onSubmit, isOpen, onClose }: any) => {
  const focusInputRef = useRef<any>(null);
  const [formState, setFormState] = useState(mergeEmptyDict);

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
    if (await onSubmit(formState)) setFormState(mergeEmptyDict);
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
            className={'form-row'}
            type={'number'}
            handleInputChange={handleInputChange} />
        )}

        <div className='form-row'>
          <button className='basic-button' type='submit'>MERGE</button>
        </div>
      </form>
    </Modal>
  );
};

MergeComicModal.propTypes = {
  onSubmit: PropTypes.func.isRequired,
  isOpen: PropTypes.bool.isRequired,
  onClose: PropTypes.func.isRequired
};

export default MergeComicModal;