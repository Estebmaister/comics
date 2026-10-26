import React, { lazy, Suspense, useState } from 'react';
import { apiErrorMessage, authorizedFetch } from '../../../api/client';
import config from '../../../util/Config';
import Loaders from '../../Loaders';
import { useToast } from '../../Toast/ToastProvider';
import { RailActionButton } from '../Actions/FloatingActionRail';
import type { CreateComicFormState } from '../types';

const SERVER = config.SERVER;
const CreateComicModal = lazy(() => import('../Modals/CreateModal'));

export const createComic = async (
  comic: CreateComicFormState,
  server = SERVER,
): Promise<string> => {
  const payload = {
    ...comic,
    last_update: new Date().getTime(),
    titles: [comic.title],
  };

  try {
    const response = await authorizedFetch(
      '/comics',
      {
        method: 'POST',
        body: JSON.stringify(payload),
      },
      server,
    );
    const data = await response.json().catch(() => null);
    if (!response.ok) {
      return apiErrorMessage(data, `Create request failed (${response.status})`);
    }
    return '';
  } catch (error) {
    return error instanceof Error ? error.message : 'Unexpected network error';
  }
};

interface CreateComicProps {
  onSuccess?: () => void;
}

const CreateComic = ({ onSuccess }: CreateComicProps) => {
  const [isCreateComicModalOpen, setIsCreateComicModalOpen] = useState(false);
  const [comicFormData, setComicFormData] = useState<CreateComicFormState | null>(null);
  const toast = useToast();

  const handleOpenCreateComicModal = () => {
    setIsCreateComicModalOpen(true);
  };

  const handleCloseCreateComicModal = () => {
    setIsCreateComicModalOpen(false);
  };

  const handleFormSubmit = async (data: CreateComicFormState) => {
    setComicFormData(data);
    const resultMsg = await createComic(data);

    if (resultMsg === '') {
      handleCloseCreateComicModal();
      toast.success({
        title: 'Comic created',
        description: `${data.title} is ready to track.`,
      });
      onSuccess?.();
      return true;
    }

    toast.error({
      title: 'Create failed',
      description: resultMsg || `Unable to create ${data.title}.`,
    });
    return false;
  };

  return (<>
    <RailActionButton
      eyebrow="Add"
      title="Create"
      description={comicFormData?.title ? `Last: ${comicFormData.title}` : 'Add a comic manually'}
      tone="cool"
      onClick={handleOpenCreateComicModal}
      aria-label="Create comic"
    />

    {isCreateComicModalOpen ? (
      <Suspense fallback={<Loaders selector="line-fw" />}>
        <CreateComicModal
          isOpen={isCreateComicModalOpen}
          onSubmit={handleFormSubmit}
          onClose={handleCloseCreateComicModal}
        />
      </Suspense>
    ) : null}
  </>);
};

export default CreateComic;
