import { SetURLSearchParams } from 'react-router-dom';

export interface Comic {
  id: number;
  title?: string;
  current_chap?: number;
  viewed_chap?: number;
  track?: boolean;
  [key: string]: any;
}

export interface PaginationState {
  total?: number;
  totalPages?: number;
  currentPage?: number;
  [key: string]: number | undefined;
}

export interface PaginationData {
  from: number;
  limit: number;
  setSearchParams: SetURLSearchParams;
  onFirstPage: boolean;
  onLastPage: boolean;
  currentPage: number;
  totalPages: number;
}

export interface ConditionalButtonProps {
  showFlag?: boolean;
  onClick: () => void;
  condFlag?: boolean;
  disabled?: boolean;
  positiveMsg: string;
  negativeMsg: string;
  className: string;
  extraClass?: string;
}
