import { ChangeEvent } from 'react';
import { SetURLSearchParams } from 'react-router-dom';
import { BUTTON_TEXT, COMIC_SEARCH_PLACEHOLDER, CSS_CLASSES } from '../constants';
import { handleOnlyTracked, handleOnlyUnchecked, handleSearchInput } from '../utils';
import { PaginationData } from '../types';
import PaginationButtons from '../PaginationButtons';

interface NavigationBarProps {
  onlyTracked: boolean;
  onlyUnchecked: boolean;
  total: number;
  queryFilter: string;
  setSearchParams: SetURLSearchParams;
  paginationData: PaginationData;
}

interface ConditionalButtonProps {
  showFlag?: boolean;
  onClick: () => void;
  condFlag?: boolean;
  disabled?: boolean;
  positiveMsg: string;
  negativeMsg: string;
  className: string;
  extraClass?: string;
}

const ConditionalButton: React.FC<ConditionalButtonProps> = ({
  showFlag = true,
  onClick,
  condFlag = false,
  disabled = false,
  positiveMsg = '',
  negativeMsg,
  className = '',
  extraClass = ''
}) => {
  if (!showFlag) return null;

  return (
    <button
      className={`${className}${condFlag ? ` ${extraClass}` : ''}`}
      onClick={onClick}
      disabled={disabled}
    >
      {condFlag ? positiveMsg : negativeMsg}
    </button>
  );
};

export const NavigationBar: React.FC<NavigationBarProps> = ({
  onlyTracked,
  onlyUnchecked,
  total,
  queryFilter,
  setSearchParams,
  paginationData
}) => {
  const handleInputChange = (e: ChangeEvent<HTMLInputElement>) => {
    handleSearchInput(setSearchParams, e?.target?.value);
  };

  return (
    <div className={CSS_CLASSES.navBar}>
      <ConditionalButton
        condFlag={onlyTracked}
        extraClass={CSS_CLASSES.reverseButton}
        onClick={handleOnlyTracked(setSearchParams, onlyTracked)}
        positiveMsg={BUTTON_TEXT.all(total)}
        negativeMsg={BUTTON_TEXT.tracked(total)}
        className={`${CSS_CLASSES.basicButton} ${CSS_CLASSES.allTrackButton}`}
      />
      
      <ConditionalButton
        showFlag={onlyTracked}
        condFlag={onlyUnchecked}
        onClick={handleOnlyUnchecked(setSearchParams, onlyUnchecked)}
        positiveMsg={BUTTON_TEXT.noFilter}
        negativeMsg={BUTTON_TEXT.unchecked}
        className={`${CSS_CLASSES.basicButton} ${CSS_CLASSES.barButton}`}
        extraClass={CSS_CLASSES.reverseButton}
      />

      <input
        className={CSS_CLASSES.searchBox}
        placeholder={COMIC_SEARCH_PLACEHOLDER}
        type="text"
        value={queryFilter}
        onChange={handleInputChange}
      />

      <PaginationButtons pagD={paginationData} />
    </div>
  );
};
