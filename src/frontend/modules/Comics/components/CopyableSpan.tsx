import React from 'react';

const CopyableSpan = ({
  copyText, text, className,
  onMouseOver, onFocus, onMouseOut, onBlur
}:
  {
    copyText: string, text?: string, className?: string,
    onMouseOver?: () => void, onFocus?: () => void, onMouseOut?: () => void, onBlur?: () => void
  }) => {
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(copyText);
    } catch (err) {
      console.error(`Failed to copy (${copyText}): `, err);
    }
  };

  return (
    <span
      onClick={handleCopy}
      className={className}
      style={{ cursor: 'pointer' }}
      onMouseOver={onMouseOver}
      onFocus={onFocus}
      onMouseOut={onMouseOut}
      onBlur={onBlur}
    >
      {text}
    </span>
  );
};

export default CopyableSpan