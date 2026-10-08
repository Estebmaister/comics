import { ChangeEvent, memo, useCallback, useMemo, useState } from 'react';
import { SetURLSearchParams } from 'react-router-dom';
import { BUTTON_TEXT, COMIC_SEARCH_PLACEHOLDER } from '../constants';
import { handleOnlyTracked, handleOnlyUnchecked } from '../utils';
import { PaginationData } from '../types';
import PagButtons from './PagButtons';
import { SortFilterModal, type SortFilterState } from './SortFilterModal';
import { AuthNav } from '../../Auth/AuthNav';

interface NavBarProps {
  onlyTracked: boolean;
  onlyUnchecked: boolean;
  total: number;
  queryFilter: string;
  onQueryFilterChange: (value: string) => void;
  setSearchParams: SetURLSearchParams;
  paginationData: PaginationData;
  ratingMin?: number;
  ratingMax?: number;
  sortBy?: string;
  sortDir?: string;
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

const NavBarComponent: React.FC<NavBarProps> = ({
  onlyTracked,
  onlyUnchecked,
  total,
  queryFilter,
  onQueryFilterChange,
  setSearchParams,
  paginationData,
  ratingMin,
  ratingMax,
  sortBy,
  sortDir,
}) => {
  const handleInputChange = useCallback((e: ChangeEvent<HTMLInputElement>) => {
    onQueryFilterChange(e?.target?.value ?? '');
  }, [onQueryFilterChange]);

  const [isSortFilterOpen, setIsSortFilterOpen] = useState(false);
  const initialSortFilterState = useMemo((): SortFilterState => ({
    ratingMin,
    ratingMax,
    sortBy: sortBy === 'rating' || sortBy === 'id' ? sortBy : 'last_update',
    sortDir: sortDir === 'asc' ? 'asc' : 'desc',
  }), [ratingMax, ratingMin, sortBy, sortDir]);

  const filterLabelParts = useMemo(() => {
    const parts: string[] = [];
    if (ratingMin !== undefined) parts.push(`R≥${ratingMin}`);
    if (ratingMax !== undefined) parts.push(`R≤${ratingMax}`);
    if (sortBy === 'rating') parts.push(`Sort:Rating`);
    else if (sortBy === 'id') parts.push(`Sort:ID`);
    return parts;
  }, [ratingMax, ratingMin, sortBy]);

  return (
    <header className="fixed inset-x-0 top-0 z-40">
      <div className="mx-auto w-full max-w-[1560px] px-2.5 pt-2 sm:px-3.5 sm:pt-1.5 lg:px-5">
        <div className="app-toolbar px-2.5 py-2 sm:px-3 sm:py-1.5">
          <div className="flex flex-col gap-1.5 sm:flex-row sm:flex-wrap sm:items-center sm:gap-1.5">
            <div className={`${onlyTracked ? 'grid grid-cols-2' : 'flex'} w-full gap-1.5 sm:flex sm:w-auto`}>
              <ConditionalButton
                condFlag={onlyTracked}
                extraClass="reverse-button"
                onClick={handleOnlyTracked(setSearchParams, onlyTracked)}
                positiveMsg={BUTTON_TEXT.all(total)}
                negativeMsg={BUTTON_TEXT.tracked(total)}
                className="basic-button w-full min-h-[2.4rem] min-w-0 px-3 text-[0.72rem] leading-none sm:w-auto sm:min-h-[2.2rem] sm:min-w-[7.2rem] sm:px-3 sm:text-[0.75rem]"
              />

              <ConditionalButton
                showFlag={onlyTracked}
                condFlag={onlyUnchecked}
                onClick={handleOnlyUnchecked(setSearchParams, onlyUnchecked)}
                positiveMsg={BUTTON_TEXT.noFilter}
                negativeMsg={BUTTON_TEXT.unchecked}
                className="basic-button w-full min-h-[2.4rem] min-w-0 px-3 text-[0.72rem] leading-none sm:w-auto sm:min-h-[2.2rem] sm:min-w-[5.8rem] sm:px-3 sm:text-[0.75rem]"
                extraClass="reverse-button"
              />
            </div>

            <input
              className="app-field px-3.5 py-2 text-sm font-medium sm:flex-1 sm:min-h-[2.4rem] sm:min-w-[220px] sm:py-1.5 md:text-base"
              placeholder={COMIC_SEARCH_PLACEHOLDER}
              type="text"
              value={queryFilter}
              onChange={handleInputChange}
            />

            <div className="grid w-full grid-cols-2 gap-1.5 sm:ml-auto sm:flex sm:w-auto sm:items-center sm:gap-1.5">
              <AuthNav />
              <button
                className="basic-button neutral-button w-full min-h-[2.4rem] min-w-0 px-3 text-[0.72rem] leading-none sm:w-auto sm:min-h-[2.2rem] sm:min-w-[6rem] sm:px-3 sm:text-[0.75rem]"
                type="button"
                onClick={() => setIsSortFilterOpen(true)}
                aria-label="Open sort and filter options"
              >
                Sort/Filter{filterLabelParts.length ? ` (${filterLabelParts.join(' · ')})` : ''}
              </button>
              <PagButtons pagD={paginationData} />
            </div>
          </div>
        </div>
      </div>

      {isSortFilterOpen ? (
        <SortFilterModal
          isOpen={isSortFilterOpen}
          onClose={() => setIsSortFilterOpen(false)}
          setSearchParams={setSearchParams}
          initialState={initialSortFilterState}
        />
      ) : null}
    </header>
  );
};

export const NavBar = memo(NavBarComponent);
