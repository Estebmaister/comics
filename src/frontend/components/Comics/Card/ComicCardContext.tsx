import { createContext, useContext } from 'react';
import type { Dispatch, SetStateAction, ReactNode } from 'react';

type ComicCardContextValue = {
  comic: Record<string, any>;
  setComic: Dispatch<SetStateAction<Record<string, any>>>;
  setViewedChap: Dispatch<SetStateAction<number>>;
};

const ComicCardContext = createContext<ComicCardContextValue | null>(null);

export const ComicCardProvider = ({
  comic,
  setComic,
  setViewedChap,
  children,
}: ComicCardContextValue & { children: ReactNode }) => (
  <ComicCardContext.Provider value={{ comic, setComic, setViewedChap }}>
    {children}
  </ComicCardContext.Provider>
);

export const useComicCard = () => {
  const context = useContext(ComicCardContext);
  if (!context) {
    throw new Error('useComicCard must be used within ComicCardProvider');
  }
  return context;
};
