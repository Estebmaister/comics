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
      <div className="mx-auto w-full max-w-[1560px] px-3 py-2.5 sm:px-4 sm:py-2 lg:px-6">
        <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center sm:gap-2">
          <div className={`${onlyTracked ? 'grid grid-cols-2' : 'flex'} w-full gap-2 sm:flex sm:w-auto`}>
            <ConditionalButton
              condFlag={onlyTracked}
              extraClass="reverse-button"
              onClick={handleOnlyTracked(setSearchParams, onlyTracked)}
              positiveMsg={BUTTON_TEXT.all(total)}
              negativeMsg={BUTTON_TEXT.tracked(total)}
              className="basic-button w-full min-h-[2.5rem] min-w-0 px-3 text-[0.72rem] leading-none sm:w-auto sm:min-h-[2.3rem] sm:min-w-[8.5rem] sm:px-3.5 sm:text-[0.76rem]"
            />

            <ConditionalButton
              showFlag={onlyTracked}
              condFlag={onlyUnchecked}
              onClick={handleOnlyUnchecked(setSearchParams, onlyUnchecked)}
              positiveMsg={BUTTON_TEXT.noFilter}
              negativeMsg={BUTTON_TEXT.unchecked}
              className="basic-button w-full min-h-[2.5rem] min-w-0 px-3 text-[0.72rem] leading-none sm:w-auto sm:min-h-[2.3rem] sm:min-w-[6.2rem] sm:px-3.5 sm:text-[0.76rem]"
              extraClass="reverse-button"
            />
          </div>

          <input
            className="w-full rounded-2xl border border-slate-200/10 bg-slate-900/75 px-4 py-2.5 text-sm font-medium text-slate-100 outline-none transition placeholder:text-slate-500 focus:border-cyan-400/60 focus:bg-slate-900 focus:ring-4 focus:ring-cyan-400/10 sm:flex-1 sm:min-h-[2.55rem] sm:min-w-[250px] sm:py-2 md:text-base"
            placeholder={COMIC_SEARCH_PLACEHOLDER}
            type="text"
            value={queryFilter}
            onChange={handleInputChange}
          />

          <div className="w-full sm:ml-auto sm:w-auto">
            <PagButtons pagD={paginationData} />
          </div>
        </div>
      </div>
    </header>
  );
};
