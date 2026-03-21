import { Dispatch, SetStateAction, useEffect, useRef, ChangeEvent, FormEvent } from 'react';
import './Modal.css';
import Modal from '../../Modal';
import InputDiv from './InputDiv';
import { ComicModalProps } from './helpers';
import type { MergeComicFormState } from '../types';

const mergeEmptyDict: MergeComicFormState = {
  baseID: 0,
  mergingID: 0,
};

interface MergeComicModalProps extends ComicModalProps<MergeComicFormState> {
  formState: MergeComicFormState;
  onFormStateChange: Dispatch<SetStateAction<MergeComicFormState>>;
}

const MergeComicModal: React.FC<MergeComicModalProps> = ({
  formState,
  onFormStateChange,
  onSubmit,
  isOpen,
  onClose,
}) => {
  const focusInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (isOpen && focusInputRef.current) {
      setTimeout(() => { focusInputRef.current?.focus(); }, 0);
    }
  }, [isOpen]);

  const handleInputChange = (
    event: ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>
  ) => {
    const { name, value, type } = event.target;
    let newEntry: string | number | string[];
    if (type === 'select-one' || type === 'number') newEntry = value === '' ? 0 : parseInt(value, 10);
    else if (type === 'checkbox') newEntry = value;
    else newEntry = value;
    onFormStateChange((prevFormData) => ({
      ...prevFormData,
      [name]: newEntry,
    }));
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (await onSubmit(formState)) onFormStateChange(mergeEmptyDict);
  };

  return (
    <Modal hasCloseBtn={true} isOpen={isOpen} onClose={onClose} size="compact">
      <form className='comic-modal-form compact-form' onSubmit={handleSubmit}>
        <header className='modal-form-header'>
          <h2>Merge Comics</h2>
          <p>Keep the base record on the left and move the duplicate into it. This dialog stays compact because the task should be fast and deliberate.</p>
        </header>

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

        <div className='form-actions'>
          <button className='basic-button' type='submit'>MERGE</button>
        </div>
      </form>
    </Modal>
  );
};

export default MergeComicModal;
