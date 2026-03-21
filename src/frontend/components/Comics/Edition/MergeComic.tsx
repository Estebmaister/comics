import React, { useState } from 'react';
import MergeComicModal from '../Modals/MergeModal';
import config from '../../../util/Config';
import { useToast } from '../../Toast/ToastProvider';
import { RailActionButton } from '../Actions/FloatingActionRail';
import type { MergeComicFormState } from '../types';

const SERVER = config.SERVER;
const mergeComic = async (
  baseID: number,
  mergingID: number,
  server = SERVER
): Promise<string> => {
  let msg = '';
  await fetch(`${server}/comics/${baseID}/${mergingID}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
  })
    .then((response) => response.json())
    .then((data) => {
      console.debug(data);
      if (data?.message !== undefined) msg = data.message;
    })
    .catch((err) => {
      console.debug(err.message);
      msg = err.message;
    });
  return msg;
};

interface MergeComicProps {
  onSuccess?: () => void;
}

const MergeComic = ({ onSuccess }: MergeComicProps) => {
  const [isMergeComicModalOpen, setIsMergeComicModalOpen] = useState(false);
  const [comicFormData, setComicFormData] = useState<MergeComicFormState | null>(null);
  const toast = useToast();

  const handleOpenMergeComicModal = () => {
    setIsMergeComicModalOpen(true);
  };

  // Set the modal boolean to false as a function to be passed
  const handleCloseMergeComicModal = () => {
    setIsMergeComicModalOpen(false);
  };

  // Send information to the server and renders a msg from response
  const handleFormSubmit = async (data: MergeComicFormState) => {
    setComicFormData(data);
    const resultMsg = await mergeComic(data?.baseID, data?.mergingID);
    if (resultMsg === '') {
      handleCloseMergeComicModal();
      toast.success({
        title: 'Comics merged',
        description: `Merged comic ${data.mergingID} into ${data.baseID}.`,
      });
      onSuccess?.();
      return true;
    }

    toast.error({
      title: 'Merge failed',
      description: resultMsg || `Unable to merge ${data.baseID} and ${data.mergingID}.`,
    });
    return false;
  };

  return (<>
    <RailActionButton
      eyebrow="Combine"
      title="Merge"
      description={comicFormData ? `${comicFormData.baseID} <- ${comicFormData.mergingID}` : 'Merge duplicate records'}
      tone="warm"
      onClick={handleOpenMergeComicModal}
      aria-label="Merge comics"
    />

    {isMergeComicModalOpen ? (
      <MergeComicModal
        isOpen={isMergeComicModalOpen}
        onSubmit={handleFormSubmit}
        onClose={handleCloseMergeComicModal}
      />
    ) : null}
  </>);
};

export default MergeComic;
