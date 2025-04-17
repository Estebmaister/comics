import { ChangeEvent, SetStateAction, Dispatch } from 'react';

export interface ComicModalProps {
  comic?: Record<string, any>;
  isOpen: boolean;
  onSubmit: (comic: Record<string, any>) => Promise<boolean>;
  onClose: () => void;
};

export function handleInputChange<T extends Record<string, any>>(
  setFormState: Dispatch<SetStateAction<T>>
) {
  return (
    event: ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement> | Record<string, any>
  ) => {
    const { name, value, selectedOptions, checked, type } = event.target;
    let newEntry: string | number | boolean | number[];
    if (type === 'select-one') newEntry = parseInt(value);
    else if (type === 'checkbox') newEntry = checked;
    else if (type === 'select-multiple') newEntry = Object
      .values(selectedOptions)?.map((options: any) => +options.value);
    else newEntry = value;
    setFormState(prevState => ({
      ...prevState,
      [name]: newEntry,
    }));
  };
};
