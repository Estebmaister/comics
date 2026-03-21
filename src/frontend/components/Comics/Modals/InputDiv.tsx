import { ChangeEvent, JSX } from 'react';

const capitalize = (str: string) => str[0].toUpperCase() + str.slice(1);

interface InputDivProps {
  focusInputRef?: React.RefObject<HTMLInputElement | null>;
  type: string;
  field: string;
  value: unknown;
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
  let normalizedValue = value;
  if (field === 'titles' && Array.isArray(normalizedValue)) normalizedValue = normalizedValue.join('|');
  const fieldValue = Array.isArray(normalizedValue)
    ? normalizedValue.map((entry) => String(entry))
    : typeof normalizedValue === 'boolean'
      ? (normalizedValue ? 'true' : 'false')
      : typeof normalizedValue === 'string' || typeof normalizedValue === 'number'
        ? normalizedValue
        : '';
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
          value={fieldValue}
          checked={normalizedValue === 'true' || normalizedValue === true}
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
          value={fieldValue}
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
            value={fieldValue}
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
          value={fieldValue}
          onChange={handleInputChange}
        />
      }
    </div>
  );
};

export default InputDiv
