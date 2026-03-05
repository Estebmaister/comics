import { JSX } from 'react';
import { Comic } from '../types';
import ComicCard from './ComicCard';
import LoadMsgs from '../../Loaders/LoadMsgs';

interface ComicsListProps {
  comics: Comic[];
  loadMsg: string | JSX.Element;
  queryFilter: string;
  onCheckoutSuccess?: () => void;
  onDeleteSuccess?: () => void;
}

const ComicsList: React.FC<ComicsListProps> = ({
  comics,
  loadMsg,
  queryFilter,
  onCheckoutSuccess,
  onDeleteSuccess,
}) => {
  if (comics.length === 0) {
    return (
      <h1 className="server">
        {loadMsg || LoadMsgs.empty(queryFilter)}
      </h1>
    );
  }

  return (
    <section className="px-2.5 pb-8 pt-28 sm:px-4 sm:pt-32 lg:px-6 xl:px-8">
      <ul className="mx-auto flex w-full max-w-[1620px] flex-wrap justify-center gap-3 sm:gap-4 xl:gap-5">
        {comics.map((comic) => (
          <ComicCard
            comic={comic}
            key={comic.id}
            onCheckoutSuccess={onCheckoutSuccess}
            onDeleteSuccess={onDeleteSuccess}
          />
        ))}
      </ul>
    </section>
  );
};

export default ComicsList;
