import { ChangeEvent } from "react";

const capitalize = (str: string) => str[0].toUpperCase() + str.slice(1);

interface InputDivProps {
  focusInputRef?: React.RefObject<HTMLInputElement>;
  type: string;
  field: string;
  value: string | number | string[] | boolean;
  handleInputChange: (event: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => void;
  selectOptDict?: Record<string, string[]>;
  multiple?: boolean;
  className?: string;
  required?: boolean;
  focus?: boolean;
}

const InputDiv = (
  { focusInputRef, type, field, value, handleInputChange, required, focus,
    selectOptDict, multiple, className = 'form-row' }: InputDivProps
): JSX.Element | null => {
  const fieldTitle = capitalize(field).split('_').join(' ');
  if (field === 'titles' && typeof (value) === 'object') value = value.join('|');
  if (type === 'none') return null;
  return (
    <div className={className}>
      <label htmlFor={field}> {fieldTitle} </label>
      {type === 'select' ?
        <select
          id={field}
          name={field}
          value={typeof value === 'boolean' ? (value ? 'true' : 'false') : value}
          onChange={handleInputChange}
          multiple={multiple}
        >
          {selectOptDict?.[field]?.map((opt: string, i: number) =>
            <option value={i} key={opt}> {opt} </option>
          )}
        </select> :
        <input
          ref={focus ? focusInputRef : undefined}
          required={required}
          min={0}
          max={field === 'rating' ? 5 : undefined}
          type={type}
          id={field}
          name={field}
          value={typeof value === 'boolean' ? (value ? 'true' : 'false') : value}
          checked={type === 'checkbox' ? (value === 'true' || value === true) : undefined}
          onChange={handleInputChange}
        />
      }
    </div>
  );
};

export default InputDiv