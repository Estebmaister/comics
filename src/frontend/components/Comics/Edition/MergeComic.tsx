import React, { useState } from 'react';
import { apiErrorMessage, authorizedFetch } from '../../../api/client';
import MergeComicModal from '../Modals/MergeModal';
import config from '../../../util/Config';
import { useToast } from '../../Toast/ToastProvider';
import { RailActionButton } from '../Actions/FloatingActionRail';
import type { MergeComicFormState } from '../types';

const SERVER = config.SERVER;
const emptyMergeComicFormState: MergeComicFormState = {
  baseID: 0,
  mergingID: 0,
};

export const mergeComic = async (
  baseID: number,
  mergingID: number,
  server = SERVER,
): Promise<string> => {
  try {
    const response = await authorizedFetch(
      `/comics/${baseID}/${mergingID}`,
      { method: 'PATCH' },
      server,
    );
    const contentType = response.headers.get('content-type') ?? '';
    const payload = contentType.includes('application/json')
      ? await response.json()
      : await response.text();

    if (!response.ok) {
      return apiErrorMessage(payload, `Merge request failed (${response.status})`);
    }
    return '';
  } catch (error) {
    return error instanceof Error ? error.message : 'Unexpected network error';
  }
};

export type MergeComicSuccessDetail = {
  baseID: number;
  mergingID: number;
};

interface MergeComicProps {
  onSuccess?: (detail: MergeComicSuccessDetail) => void;
}

const MergeComic = ({ onSuccess }: MergeComicProps) => {
  const [isMergeComicModalOpen, setIsMergeComicModalOpen] = useState(false);
  const [comicFormData, setComicFormData] = useState<MergeComicFormState>(emptyMergeComicFormState);
  const [submitError, setSubmitError] = useState('');
  const toast = useToast();

  const handleOpenMergeComicModal = () => {
    setSubmitError('');
    setIsMergeComicModalOpen(true);
  };

  // Set the modal boolean to false as a function to be passed
  const handleCloseMergeComicModal = () => {
    setSubmitError('');
    setIsMergeComicModalOpen(false);
  };

  // Send information to the server and renders a msg from response
  const handleFormSubmit = async (data: MergeComicFormState) => {
    setSubmitError('');
    setComicFormData(data);
    const resultMsg = await mergeComic(data?.baseID, data?.mergingID);
    if (resultMsg === '') {
      setComicFormData(emptyMergeComicFormState);
      handleCloseMergeComicModal();
      toast.success({
        title: 'Comics merged',
        description: `Merged comic ${data.mergingID} into ${data.baseID}.`,
      });
      onSuccess?.({ baseID: data.baseID, mergingID: data.mergingID });
      return true;
    }

    setSubmitError(resultMsg || `Unable to merge ${data.baseID} and ${data.mergingID}.`);
    return false;
  };

  return (<>
    <RailActionButton
      eyebrow="Combine"
      title="Merge"
      description={
        comicFormData.baseID || comicFormData.mergingID
          ? `${comicFormData.baseID || 'Base'} <- ${comicFormData.mergingID || 'Duplicate'}`
          : 'Merge duplicate records'
      }
      tone="warm"
      onClick={handleOpenMergeComicModal}
      aria-label="Merge comics"
    />

    <MergeComicModal
      isOpen={isMergeComicModalOpen}
      formState={comicFormData}
      submissionError={submitError}
      onFormStateChange={setComicFormData}
      onSubmit={handleFormSubmit}
      onClose={handleCloseMergeComicModal}
    />
  </>);
};

export default MergeComic;
