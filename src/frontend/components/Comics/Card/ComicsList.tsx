import { Comic } from '../types';
import { CSS_CLASSES } from '../constants';
import ComicCard from './ComicCard';
import LoadMsgs from '../../Loaders/LoadMsgs';

interface ComicsListProps {
  comics: Comic[];
  loadMsg: string | JSX.Element;
  queryFilter: string;
}

const ComicsList: React.FC<ComicsListProps> = ({
  comics,
  loadMsg,
  queryFilter
}) => {
  if (comics.length === 0) {
    return (
      <h1 className={CSS_CLASSES.serverMessage}>
        {loadMsg || LoadMsgs.empty(queryFilter)}
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

export default ComicsList;