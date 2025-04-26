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
    <span
      onClick={() => handleCopyToClipboard(textToCopy)}
      className={className}
      style={{ cursor: 'pointer' }}
      onMouseOver={onMouseOver}
      onFocus={onFocus}
      onMouseOut={onMouseOut}
      onBlur={onBlur}
    >
      {textToShow}
    </span>
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