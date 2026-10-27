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
  submissionError?: string;
  onFormStateChange: Dispatch<SetStateAction<MergeComicFormState>>;
}

const MergeComicModal: React.FC<MergeComicModalProps> = ({
  formState,
  submissionError,
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
    <Modal hasCloseBtn={true} isOpen={isOpen} onClose={onClose} size="default">
      <form className="comic-modal-form compact-form merge-modal-form" onSubmit={handleSubmit}>
        <header className='modal-form-header'>
          <h2>Merge Comics</h2>
          <p>
            Enter the record to keep (base ID) and the duplicate to fold in. Use Swap IDs
            if you want the higher-ID row to survive.
          </p>
        </header>

        {submissionError ? (
          <p className='form-feedback form-feedback-error' role='alert'>
            {submissionError}
          </p>
        ) : null}

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

        <div className="merge-modal-footer form-actions">
          <button
            type="button"
            className="basic-button reverse-button"
            onClick={() => onFormStateChange((prev) => ({
              baseID: prev.mergingID,
              mergingID: prev.baseID,
            }))}
            disabled={!formState.baseID || !formState.mergingID}
          >
            Swap IDs
          </button>
          <button className="basic-button" type="submit">MERGE</button>
        </div>
      </form>
    </Modal>
  );
};

export default MergeComicModal;
