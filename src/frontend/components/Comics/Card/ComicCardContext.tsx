import { createContext, useContext, useMemo } from 'react';
import type { Dispatch, SetStateAction, ReactNode } from 'react';
import type { Comic } from '../types';

type ComicCardContextValue = {
  comic: Comic;
  setComic: Dispatch<SetStateAction<Comic>>;
  setViewedChap: Dispatch<SetStateAction<number>>;
};

const ComicCardContext = createContext<ComicCardContextValue | null>(null);

export const ComicCardProvider = ({
  comic,
  setComic,
  setViewedChap,
  children,
}: ComicCardContextValue & { children: ReactNode }) => {
  const value = useMemo(
    () => ({ comic, setComic, setViewedChap }),
    [comic, setComic, setViewedChap]
  );

  return (
    <ComicCardContext.Provider value={value}>
      {children}
    </ComicCardContext.Provider>
  );
};

export const useComicCard = () => {
  const context = useContext(ComicCardContext);
  if (!context) {
    throw new Error('useComicCard must be used within ComicCardProvider');
  }
  return context;
};
