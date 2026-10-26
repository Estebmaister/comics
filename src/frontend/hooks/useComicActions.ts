import { useCallback } from 'react';
import type { Dispatch, SetStateAction } from 'react';
import { checkoutComic, delComic, trackComic } from '../util/ServerHelpers';
import type { Comic } from '../components/Comics/types';

type UseComicActionsArgs = {
  comicId: number;
  currentChap: number;
  isTracked: boolean;
  setComic: Dispatch<SetStateAction<Comic>>;
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
    void checkoutComic(currentChap, comicId, setCheck, setViewedChap, onCheckoutSuccess);
  }, [comicId, currentChap, onCheckoutSuccess, setCheck, setViewedChap]);

  const handleTrackToggle = useCallback(() => {
    void trackComic(isTracked, comicId, setTrackComic);
  }, [comicId, isTracked, setTrackComic]);

  const handleDelete = useCallback(() => {
    void delComic(comicId, setDel, onDeleteSuccess);
  }, [comicId, onDeleteSuccess, setDel]);

  return {
    handleCheckout,
    handleTrackToggle,
    handleDelete,
  };
};
