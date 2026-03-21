import React from 'react';

interface CopyableSpanProps {
  textToCopy: string | number,
  textToShow?: string,
  className?: string,
  ariaLabel?: string,
}

const CopyableSpan: React.FC<CopyableSpanProps> = ({
  textToCopy, textToShow, className, ariaLabel,
}) => {
  return (
    <button
      type="button"
      onClick={() => handleCopyToClipboard(String(textToCopy))}
      className={className}
      aria-label={ariaLabel ?? `Copy ${textToCopy}`}
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
