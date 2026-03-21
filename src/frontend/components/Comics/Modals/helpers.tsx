import { ChangeEvent, SetStateAction, Dispatch } from 'react';
import type { Comic } from '../types';

export interface ComicModalProps<T> {
  comic?: Comic;
  isOpen: boolean;
  onSubmit: (comic: T) => Promise<boolean>;
  onClose: () => void;
};

export function handleInputChange<T extends object>(
  setFormState: Dispatch<SetStateAction<T>>
) {
  return (
    event: ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>
  ) => {
    const target = event.target;
    const { name, value, type } = target;
    let newEntry: string | number | boolean | number[];
    if (type === 'select-one') newEntry = parseInt(value);
    else if (type === 'checkbox' && target instanceof HTMLInputElement) newEntry = target.checked;
    else if (type === 'select-multiple' && target instanceof HTMLSelectElement) {
      newEntry = Array.from(target.selectedOptions).map((option) => +option.value);
    }
    else newEntry = value;
    setFormState(prevState => ({ ...prevState, [name]: newEntry, }));
  };
};
