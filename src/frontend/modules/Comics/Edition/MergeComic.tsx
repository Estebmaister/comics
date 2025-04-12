import React, { useState } from 'react';
import MergeComicModal from '../Modals/MergeModal';
import './MergeComic.css';
import config from '../../../util/Config';
import OpMsg from './OpMsg';

const SERVER = config.SERVER;
const mergeComic: (baseID: number, mergingID: number, server?: string) => Promise<string> = async (baseID: number, mergingID: number, server = SERVER) => {
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
}

const MergeComic = () => {
  const [isMergeComicModalOpen, setIsMergeComicModalOpen] = useState(false);
  const [comicFormData, setComicFormData] = useState<any>(null);
  const [msg, setMsg] = useState('');
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
    setHideMsg(false);
    setShowMsg(true);

    const resultMsg = await mergeComic(data?.baseID, data?.mergingID);
    setMsg(resultMsg);
    if (resultMsg === '') {
      handleCloseMergeComicModal();
      setFailMsg(false);
      return true;
    }
    setFailMsg(true);
    return false;
  };

  return (<>
    <button className={'button-merge'} onClick={handleOpenMergeComicModal}>
    </button>

    <OpMsg
      comicType={comicFormData?.baseID + ' &'}
      title={comicFormData?.mergingID}
      msg={msg}
      operation="merging"
      showMsg={showMsg}
      hideMsg={hideMsg}
      failMsg={failMsg}
      setHideMsg={setHideMsg}
    />

    <MergeComicModal
      isOpen={isMergeComicModalOpen}
      onSubmit={handleFormSubmit}
      onClose={handleCloseMergeComicModal}
    />
  </>);
};

export default MergeComic;