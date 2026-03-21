import { lazy, Suspense, useState } from 'react';
import type { ReactNode } from 'react';
import config from '../../../util/Config';
import db_classes from '../../../../db/db_classes.json';
import { useComicCard } from '../Card/ComicCardContext';
import Loaders from '../../Loaders';
import { useToast } from '../../Toast/ToastProvider';
import type { Comic } from '../types';
import './EditComic.css';

const SERVER = config.SERVER;
const EditComicModal = lazy(() => import('../Modals/EditModal'));
const edit = async (
  comic: Comic,
  server = SERVER
): Promise<[Comic | undefined, string]> => {
  let newData: Comic | undefined;
  let msg = '';
  const { deleted, track, ...data } = { ...comic, last_update: new Date().getTime() };
  console.debug(JSON.stringify(data));

  await fetch(`${server}/comics/${comic.id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
    headers: { 'Content-Type': 'application/json' },
  })
    .then((response) => response.json())
    .then((responseData) => {
      console.debug(responseData);
      if (responseData?.message !== undefined) {
        newData = undefined;
        msg = responseData.message;
        return;
      }
      newData = responseData;
    })
    .catch((err) => {
      console.debug(err.message);
      newData = undefined;
      msg = err.message;
    });

  return [newData, msg];
};

interface EditComicProps {
  className?: string;
  children?: ReactNode;
}

const EditComic = ({ className, children }: EditComicProps) => {
  const { comic, setComic, setViewedChap } = useComicCard();
  const [isEditComicModalOpen, setIsEditComicModalOpen] = useState(false);
  const toast = useToast();

  const handleOpenEditComicModal = () => {
    setIsEditComicModalOpen(true);
  };

  const handleCloseEditComicModal = () => {
    setIsEditComicModalOpen(false);
  };

  const handleFormSubmit = async (data: Comic) => {
    const [newData, resultMsg] = await edit(data);

    if (newData !== undefined) {
      setComic(newData);
      setViewedChap(newData?.viewed_chap);
      handleCloseEditComicModal();
      toast.success({
        title: 'Comic updated',
        description: `${db_classes?.com_type[newData.com_type]} ${newData.titles[0]} is saved.`,
      });
      return true;
    }

    toast.error({
      title: 'Update failed',
      description: resultMsg || `Unable to update ${comic?.titles?.[0] ?? 'comic'}.`,
    });
    return false;
  };

  return (<>
    <button
      className={className ?? 'edit-button'}
      onClick={handleOpenEditComicModal}
    >
      {children ?? 'Edit'}
    </button>

    {isEditComicModalOpen ? (
      <Suspense fallback={<Loaders selector="line-fw" />}>
        <EditComicModal
          comic={comic}
          isOpen={isEditComicModalOpen}
          onSubmit={handleFormSubmit}
          onClose={handleCloseEditComicModal}
        />
      </Suspense>
    ) : null}
  </>);
};

export default EditComic;
