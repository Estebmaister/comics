import React from 'react';

interface CopyableSpanProps {
  textToCopy: string,
  textToShow?: string,
  className?: string,
  onMouseOver?: () => void,
  onFocus?: () => void,
  onMouseOut?: () => void,
  onBlur?: () => void
}

const CopyableSpan: React.FC<CopyableSpanProps> = ({
  textToCopy, textToShow, className,
  onMouseOver, onFocus, onMouseOut, onBlur
}) => {
  return (
    <button
      type="button"
      onClick={() => handleCopyToClipboard(textToCopy)}
      className={className}
      onMouseOver={onMouseOver}
      onFocus={onFocus}
      onMouseOut={onMouseOut}
      onBlur={onBlur}
    >
      {textToShow}
    </button>
  );
};

export default CopyableSpan

const handleCopyToClipboard = async (textToCopy: string) => {
  try {
    await navigator.clipboard.writeText(textToCopy);
  } catch (err) {
    console.error(`Failed to copy to clipboard (${textToCopy}): `, err);
  }
};
