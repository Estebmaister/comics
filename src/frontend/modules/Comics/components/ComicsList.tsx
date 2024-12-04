import { Comic } from '../types';
import { ComicCard } from '../Card/ComicCard';
import { loadMsgs } from '../../../util/ServerHelpers';
import { CSS_CLASSES } from '../constants';

interface ComicsListProps {
  comics: Comic[];
  loadMsg: string;
  queryFilter: string;
}

export const ComicsList: React.FC<ComicsListProps> = ({
  comics,
  loadMsg,
  queryFilter
}) => {
  if (comics.length === 0) {
    return (
      <h1 className={CSS_CLASSES.serverMessage}>
        {loadMsg || loadMsgs.empty(queryFilter)}
      </h1>
    );
  }

  return (
    <ul className={CSS_CLASSES.comicList}>
      {comics.map((comic) => (
        <ComicCard 
          comic={comic} 
          key={comic.id} 
        />
      ))}
    </ul>
  );
};
