import { lazy, Suspense, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import config from '../../../util/Config';
import { patchComic } from '../../../util/comicUpdate';
import type { CatalogMutationOptions } from '../../../util/ServerHelpers';
import db_classes from '../../../../../db/db_classes.json';
import { useComicCard } from '../Card/ComicCardContext';
import Loaders from '../../Loaders';
import { useToast } from '../../Toast/ToastProvider';
import { useDiscoveryConflict } from '../Discovery/DiscoveryConflictContext';
import type { Comic } from '../types';

const SERVER = config.SERVER;
const EditComicModal = lazy(() => import('../Modals/EditModal'));

export const editComic = async (
  comic: Comic,
  options?: CatalogMutationOptions,
  server = SERVER,
): Promise<[Comic | undefined, string]> => {
  const { deleted, track, ...payload } = comic;

  const attempt = async (): Promise<[Comic | undefined, string]> => {
    const result = await patchComic(comic.id, payload as Record<string, unknown>, server);
    if (result.ok) {
      if (!result.comic) {
        return [undefined, 'Update returned an invalid comic payload'];
      }
      return [result.comic, ''];
    }
    if (result.kind === 'conflict' && options?.onConflict) {
      options.onConflict(result.conflict, async () => {
        const [updated] = await attempt();
        return updated !== undefined;
      });
      return [undefined, ''];
    }
    if (result.kind === 'server') {
      return [undefined, result.message];
    }
    return [undefined, 'Unexpected network error'];
  };

  return attempt();
};

interface EditComicProps {
  className?: string;
  children?: ReactNode;
}

const EditComic = ({ className, children }: EditComicProps) => {
  const { comic, setComic, setViewedChap } = useComicCard();
  const discoveryConflict = useDiscoveryConflict();
  const catalogMutationOptions = useMemo<CatalogMutationOptions | undefined>(() => {
    if (!discoveryConflict) return undefined;
    return {
      onConflict: (conflict, retry) => {
        discoveryConflict.openCatalogConflict({ conflict, retry });
      },
    };
  }, [discoveryConflict]);
  const [isEditComicModalOpen, setIsEditComicModalOpen] = useState(false);
  const toast = useToast();

  const handleOpenEditComicModal = () => {
    setIsEditComicModalOpen(true);
  };

  const handleCloseEditComicModal = () => {
    setIsEditComicModalOpen(false);
  };

  const handleFormSubmit = async (data: Comic) => {
    const [newData, resultMsg] = await editComic(data, catalogMutationOptions);

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

    if (resultMsg !== '') {
      toast.error({
        title: 'Update failed',
        description: resultMsg || `Unable to update ${comic?.titles?.[0] ?? 'comic'}.`,
      });
    }
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
