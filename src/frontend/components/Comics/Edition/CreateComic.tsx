import React, { useState } from 'react';
import CreateComicModal from '../Modals/CreateModal';
import OpMsg from './OpMsg';
import './CreateComic.css';
import config from '../../../util/Config';
import db_classes from '../../../../db/db_classes.json';

const SERVER = config.SERVER;
const create: (comic: Record<string, any>, server?: string) => Promise<string> = async (comic, server = SERVER) => {
  let msg = '';
  const last_update = { last_update: new Date().getTime() }
  const titles = { titles: [comic.title] };
  const data = { ...comic, ...last_update, ...titles }
  console.debug(JSON.stringify(data))
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

const CreateComic = () => {
  const [isCreateComicModalOpen, setIsCreateComicModalOpen] = useState(false);
  const [comicFormData, setComicFormData] = useState<Record<string, any>>({});
  const [msg, setMsg] = useState('');
  const [showMsg, setShowMsg] = useState(false);
  const [hideMsg, setHideMsg] = useState(false);
  const [failMsg, setFailMsg] = useState(false);

  const handleOpenCreateComicModal = () => {
    setIsCreateComicModalOpen(true);
    setShowMsg(false);
    setFailMsg(false);
  };

  const handleCloseCreateComicModal = () => {
    setIsCreateComicModalOpen(false);
  };

  const handleFormSubmit = async (data: {}) => {
    setComicFormData(data);
    const resultMsg = await create(data);
    setMsg(resultMsg);
    setHideMsg(false);
    setShowMsg(true);

    if (resultMsg === '') {
      handleCloseCreateComicModal();
      setFailMsg(false);
      return true;
    }
    setFailMsg(true);
    return false;
  };

  return (<>
    <button className={'button-plus'} onClick={handleOpenCreateComicModal}>
    </button>

    <OpMsg
      comicType={db_classes?.com_type[comicFormData?.com_type]}
      title={comicFormData?.title}
      msg={msg}
      operation="creation"
      showMsg={showMsg}
      hideMsg={hideMsg}
      failMsg={failMsg}
      setHideMsg={setHideMsg}
    />

    <CreateComicModal
      isOpen={isCreateComicModalOpen}
      onSubmit={handleFormSubmit}
      onClose={handleCloseCreateComicModal}
    />
  </>);
};

export default CreateComic;