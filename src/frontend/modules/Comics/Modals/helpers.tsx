import { ChangeEvent, SetStateAction } from 'react';

export interface ComicModalProps {
  comic?: any;
  isOpen: boolean;
  onSubmit: (comic: any) => Promise<boolean>;
  onClose: () => void;
}

export const handleInputChange = (setFormState: { (value: SetStateAction<any>): void; }) => (
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

