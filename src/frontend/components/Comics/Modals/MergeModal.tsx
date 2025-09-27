import { useState, useEffect, useRef, ChangeEvent, FormEvent } from 'react';
import './Modal.css';
import Modal from '../../Modal';
import InputDiv from './InputDiv';
import { ComicModalProps } from './helpers';

const mergeEmptyDict = {
  baseID: 0,
  mergingID: 0,
};

const MergeComicModal: React.FC<ComicModalProps> = ({ onSubmit, isOpen, onClose }) => {
  const focusInputRef = useRef<HTMLInputElement>(null);
  const [formState, setFormState] = useState(mergeEmptyDict);

  useEffect(() => {
    if (isOpen && focusInputRef.current) {
      setTimeout(() => { focusInputRef.current?.focus(); }, 0);
    }
  }, [isOpen]);

  const handleInputChange = (event: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value, type } = event.target;
    let newEntry: string | number | string[];
    if (type === 'select-one') newEntry = parseInt(value);
    else if (type === 'checkbox') newEntry = value;
    else newEntry = value;
    setFormState((prevFormData) => ({
      ...prevFormData,
      [name]: newEntry,
    }));
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (await onSubmit(formState)) setFormState(mergeEmptyDict);
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
            className={'form-row'}
            type={'number'}
            required={true}
            focus={kField === 'baseID'}
            handleInputChange={handleInputChange} />
        )}

        <div className='form-row'>
          <button className='basic-button' type='submit'>MERGE</button>
        </div>
      </form>
    </Modal>
  );
};

export default MergeComicModal;