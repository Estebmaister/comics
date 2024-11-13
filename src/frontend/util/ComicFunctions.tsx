// Not in use

export const filterComics = (
    comics: any[], 
    filterWord: string, 
    trackedOnly: boolean, 
    uncheckedOnly: boolean
  ) => 
  comics.filter((comic: any) => {
    for (const title of comic.titles) {
      if (trackedOnly && !comic.track) {
        return false;
      }
      if (uncheckedOnly && (comic.viewed_chap === comic.current_chap)) {
        return false;
      }
      if (title.toLowerCase().includes( filterWord.toLowerCase() )) {
        return true;
      }
    }
    return false;
  }
);
