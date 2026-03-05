import { ChangeEvent, JSX } from 'react';

const capitalize = (str: string) => str[0].toUpperCase() + str.slice(1);

interface InputDivProps {
  focusInputRef?: React.RefObject<HTMLInputElement | null>;
  type: string;
  field: string;
  value: string | number | string[] | boolean;
  handleInputChange: (event: ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) => void;
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
  const rowClassName = `${className}${type === 'checkbox' ? ' checkbox-row' : ''}`;

  if (type === 'checkbox') {
    return (
      <div className={rowClassName}>
        <input
          ref={focus ? (focusInputRef as React.Ref<HTMLInputElement>) : undefined}
          required={required}
          type={type}
          id={field}
          name={field}
          value={typeof value === 'boolean' ? (value ? 'true' : 'false') : value}
          checked={value === 'true' || value === true}
          onChange={handleInputChange}
        />
        <label htmlFor={field}> {fieldTitle} </label>
      </div>
    );
  }

  return (
    <div className={rowClassName}>
      <label htmlFor={field}> {fieldTitle} </label>
      {type === 'select' ?
        <select
          id={field}
          name={field}
          value={typeof value === 'boolean' ? (value ? 'true' : 'false') : value}
          onChange={handleInputChange}
          multiple={multiple}
          size={multiple ? 4 : undefined}
        >
          {selectOptDict?.[field]?.map((opt: string, i: number) =>
            <option value={i} key={opt}> {opt} </option>
          )}
        </select> :
        type === 'textarea' ?
          <textarea
            required={required}
            id={field}
            name={field}
            value={typeof value === 'boolean' ? (value ? 'true' : 'false') : value}
            onChange={handleInputChange}
            rows={3}
          /> :
        <input
          ref={focus ? (focusInputRef as React.Ref<HTMLInputElement>) : undefined}
          required={required}
          min={0}
          max={field === 'rating' ? 5 : undefined}
          type={type}
          id={field}
          name={field}
          value={typeof value === 'boolean' ? (value ? 'true' : 'false') : value}
          onChange={handleInputChange}
        />
      }
    </div>
  );
};

export default InputDiv
