import React, { lazy, Suspense, useState } from 'react';
import config from '../../../util/Config';
import Loaders from '../../Loaders';
import { useToast } from '../../Toast/ToastProvider';
import { RailActionButton } from '../Actions/FloatingActionRail';
import type { CreateComicFormState } from '../types';

const SERVER = config.SERVER;
const CreateComicModal = lazy(() => import('../Modals/CreateModal'));
const create = async (
  comic: CreateComicFormState,
  server = SERVER
): Promise<string> => {
  let msg = '';
  const last_update = { last_update: new Date().getTime() };
  const titles = { titles: [comic.title] };
  const data = { ...comic, ...last_update, ...titles };
  console.debug(JSON.stringify(data));
  await fetch(`${server}/comics`, {
    method: 'POST',
    body: JSON.stringify(data),
    headers: { 'Content-Type': 'application/json' },
  })
    .then((response) => response.json())
    .then((data) => {
      console.debug(data);
      if (data?.message) msg = data.message;
    })
    .catch((err) => {
      console.debug(err.message);
      msg = err.message;
    });
  return msg;
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
    const resultMsg = await create(data);

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
