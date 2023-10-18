const capitalize = (string) => string[0].toUpperCase() + string.slice(1);

const InputDiv = ({
  ref, 
  type, 
  field, 
  value, 
  optDict, 
  handleInputChange, 
  className = 'form-row'
}) => {
  const fieldTitle = capitalize(field).split('_').join(' ');
  return (
    <div className={className}>
      <label htmlFor={field}> {fieldTitle} </label>
      {type === "select" ?
        <select 
          id={field} 
          name={field} 
          value={value} 
          onChange={handleInputChange}
        >
          {optDict[field].map((opt, i) => 
            <option value={i} key={opt}> {opt} </option>
          )}
        </select> :
        <input
          ref={field === 'title' ? ref : null}
          required={field === 'title'}
          type={type} 
          id={field} 
          name={field}
          value={value}
          onChange={handleInputChange}
        />    
      }
    </div>
  );
};

export default InputDiv