import { ChangeEvent } from 'react';
import { SetURLSearchParams } from 'react-router-dom';
import { BUTTON_TEXT, COMIC_SEARCH_PLACEHOLDER } from '../constants';
import { handleOnlyTracked, handleOnlyUnchecked, handleSearchInput } from '../utils';
import { PaginationData } from '../types';
import PagButtons from './PagButtons';

interface NavBarProps {
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

export const NavBar: React.FC<NavBarProps> = ({
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
    <header className="fixed inset-x-0 top-0 z-40 border-b border-slate-200/10 bg-slate-950/70 shadow-halo backdrop-blur-xl">
      <div className="mx-auto flex w-full max-w-[1560px] flex-wrap items-center gap-2.5 px-3 py-3 sm:px-4 lg:px-6">
        <ConditionalButton
          condFlag={onlyTracked}
          extraClass="reverse-button"
          onClick={handleOnlyTracked(setSearchParams, onlyTracked)}
          positiveMsg={BUTTON_TEXT.all(total)}
          negativeMsg={BUTTON_TEXT.tracked(total)}
          className="basic-button min-w-[8.75rem]"
        />

        <ConditionalButton
          showFlag={onlyTracked}
          condFlag={onlyUnchecked}
          onClick={handleOnlyUnchecked(setSearchParams, onlyUnchecked)}
          positiveMsg={BUTTON_TEXT.noFilter}
          negativeMsg={BUTTON_TEXT.unchecked}
          className="basic-button min-w-[6.5rem]"
          extraClass="reverse-button"
        />

        <input
          className="w-full flex-1 rounded-2xl border border-slate-200/10 bg-slate-900/75 px-4 py-2.5 text-sm font-medium text-slate-100 outline-none transition placeholder:text-slate-500 focus:border-cyan-400/60 focus:bg-slate-900 focus:ring-4 focus:ring-cyan-400/10 sm:w-auto sm:min-w-[250px] md:text-base"
          placeholder={COMIC_SEARCH_PLACEHOLDER}
          type="text"
          value={queryFilter}
          onChange={handleInputChange}
        />

        <PagButtons pagD={paginationData} />
      </div>
    </header>
  );
};
