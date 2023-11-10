const capitalize = (string) => string[0].toUpperCase() + string.slice(1);

const InputDiv = (
  { focusInputRef, type, field, value, selectOptDict, 
  handleInputChange, multiple, className = 'form-row' }
) => {
  const fieldTitle = capitalize(field).split('_').join(' ');
  if (field === 'titles' && typeof(value) === 'object') value = value.join('|');
  if (type === 'none') return;
  return (
    <div className={className}>
      <label htmlFor={field}> {fieldTitle} </label>
      {type === 'select' ?
        <select 
          id={field} 
          name={field} 
          value={value} 
          onChange={handleInputChange}
          multiple={multiple}
        >
          {selectOptDict[field].map((opt, i) => 
            <option value={i} key={opt}> {opt} </option>
          )}
        </select> :
        <input
          ref={field === 'title' ? focusInputRef : null}
          required={field === 'title'}
          min={0}
          type={type} 
          id={field} 
          name={field}
          value={value}
          checked={type === 'checkbox' ? value : null}
          onChange={handleInputChange}
        />    
      }
    </div>
  );
};

export default InputDiv