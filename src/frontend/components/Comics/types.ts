import { SetURLSearchParams } from 'react-router-dom';

export type ComicLookupValue = number | string;

export interface Comic {
  id: number;
  titles: string[];
  cover: string;
  cover_visible?: boolean;
  author: string;
  current_chap: number;
  viewed_chap: number;
  track: boolean;
  status: number;
  com_type: number;
  genres: ComicLookupValue[];
  published_in: ComicLookupValue[];
  description?: string;
  rating?: number;
  deleted?: boolean;
  last_update?: number;
  [key: string]: unknown;
}

export interface CreateComicFormState {
  title: string;
  track: boolean;
  current_chap: number;
  viewed_chap: number;
  cover: string;
  description: string;
  author: string;
  com_type: number;
  status: number;
  published_in: number[];
  genres: number[];
}

export interface MergeComicFormState {
  baseID: number;
  mergingID: number;
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

export interface ToastInput {
  title: string;
  description?: string;
  tone?: 'success' | 'error' | 'info';
  duration?: number;
}
