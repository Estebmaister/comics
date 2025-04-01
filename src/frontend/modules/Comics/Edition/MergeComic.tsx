import React, { useState } from 'react';
import MergeComicModal from '../Modals/MergeModal';
import './MergeComic.css';
import config from '../../../util/Config';

const SERVER = config.SERVER;
const SHOW_MESSAGE_TIMEOUT = config.SHOW_MESSAGE_TIMEOUT;
const mergeComic = async (baseID: number, mergingID: number, server = SERVER) => {
  let success = true;
  await fetch(`${server}/comics/${baseID}/${mergingID}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
  })
    .then((response) => response.json())
    .then((data) => {
      console.debug(data);
      if (data?.message !== undefined) success = false;
    })
    .catch((err) => {
      console.debug(err.message);
      success = false;
    });
  return success;
}

const MergeComic = () => {
  const [isMergeComicModalOpen, setIsMergeComicModalOpen] = useState(false);
  const [comicFormData, setComicFormData] = useState<any>(null);
  const [showMsg, setShowMsg] = useState(false);
  const [hideMsg, setHideMsg] = useState(false);
  const [failMsg, setFailMsg] = useState(false);

  const handleOpenMergeComicModal = () => {
    setIsMergeComicModalOpen(true);
    setShowMsg(false);
    setFailMsg(false);
  };

  // Set the modal boolean to false as a function to be passed
  const handleCloseMergeComicModal = () => {
    setIsMergeComicModalOpen(false);
  };

  // Send information to the server and renders a msg from response
  const handleFormSubmit = async (data: any) => {
    setComicFormData(data);

    if (await mergeComic(data?.baseID, data?.mergingID)) {
      handleCloseMergeComicModal();
      setFailMsg(false);
      setHideMsg(false);
      setShowMsg(true);
      return true;
    }
    setHideMsg(false);
    setFailMsg(true);
    setShowMsg(true);
    return false;
  };

  const timerHide = () => {
    setTimeout(() => setHideMsg(true), SHOW_MESSAGE_TIMEOUT);
    return true;
  }

  return (<>
    <button className={'button-merge'} onClick={handleOpenMergeComicModal}>
    </button>

    {(showMsg && timerHide()) && (
      <div className={
        `msg-box ${hideMsg ? 'msg-hide' : ''} ${failMsg ? 'msg-fail' : ''}`
      }>
        comics <b>{comicFormData?.baseID}</b>{' '}
        & <b>{comicFormData?.mergingID}</b> merging
        {failMsg ? ' failed, check that comics type match' : ' succeed'}.
      </div>
    )}

    <MergeComicModal
      isOpen={isMergeComicModalOpen}
      onSubmit={handleFormSubmit}
      onClose={handleCloseMergeComicModal}
    />
  </>);
};

export default MergeComic;