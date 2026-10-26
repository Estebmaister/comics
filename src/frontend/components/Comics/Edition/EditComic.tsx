import { lazy, Suspense, useState } from 'react';
import type { ReactNode } from 'react';
import { apiErrorMessage, authorizedFetch } from '../../../api/client';
import config from '../../../util/Config';
import db_classes from '../../../../../db/db_classes.json';
import { useComicCard } from '../Card/ComicCardContext';
import Loaders from '../../Loaders';
import { useToast } from '../../Toast/ToastProvider';
import type { Comic } from '../types';

const SERVER = config.SERVER;
const EditComicModal = lazy(() => import('../Modals/EditModal'));

export const editComic = async (
  comic: Comic,
  server = SERVER,
): Promise<[Comic | undefined, string]> => {
  const { deleted, track, ...payload } = comic;

  try {
    const response = await authorizedFetch(
      `/comics/${comic.id}`,
      {
        method: 'PUT',
        body: JSON.stringify(payload),
      },
      server,
    );
    const data = await response.json().catch(() => null);
    if (!response.ok) {
      return [
        undefined,
        apiErrorMessage(data, `Update request failed (${response.status})`),
      ];
    }
    if (data === null || typeof data !== 'object') {
      return [undefined, 'Update returned an invalid comic payload'];
    }
    return [data as Comic, ''];
  } catch (error) {
    return [
      undefined,
      error instanceof Error ? error.message : 'Unexpected network error',
    ];
  }
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
    const [newData, resultMsg] = await editComic(data);

    if (newData !== undefined) {
      setComic(newData);
      setViewedChap(Number(newData.viewed_chap ?? 0));
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
      className={className ?? 'basic-button neutral-button'}
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
