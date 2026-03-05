import { useCallback } from 'react';
import type { Dispatch, SetStateAction } from 'react';
import { checkoutComic, delComic, trackComic } from '../util/ServerHelpers';

type UseComicActionsArgs = {
  comicId: number;
  currentChap: number;
  isTracked: boolean;
  setComic: Dispatch<SetStateAction<Record<string, any>>>;
  setViewedChap: Dispatch<SetStateAction<number>>;
  setCheck: Dispatch<SetStateAction<boolean>>;
  setDel: Dispatch<SetStateAction<boolean>>;
  onCheckoutSuccess?: () => void;
  onDeleteSuccess?: () => void;
};

export const useComicActions = ({
  comicId,
  currentChap,
  isTracked,
  setComic,
  setViewedChap,
  setCheck,
  setDel,
  onCheckoutSuccess,
  onDeleteSuccess,
}: UseComicActionsArgs) => {
  const setTrackComic = useCallback((track: boolean) => {
    setComic((prev) => ({ ...prev, track }));
  }, [setComic]);

  const handleCheckout = useCallback(() => {
    checkoutComic(currentChap, comicId, setCheck, setViewedChap, onCheckoutSuccess);
  }, [comicId, currentChap, onCheckoutSuccess, setCheck, setViewedChap]);

  const handleTrackToggle = useCallback(() => {
    trackComic(isTracked, comicId, setTrackComic);
  }, [comicId, isTracked, setTrackComic]);

  const handleDelete = useCallback(() => {
    delComic(comicId, setDel, onDeleteSuccess);
  }, [comicId, onDeleteSuccess, setDel]);

  return {
    handleCheckout,
    handleTrackToggle,
    handleDelete,
  };
};
