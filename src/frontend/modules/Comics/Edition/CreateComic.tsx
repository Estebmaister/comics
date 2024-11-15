import React, { useState } from 'react';
import CreateComicModal from '../Modals/CreateModal';
import './CreateComic.css';
import db_classes from '../../../../db/db_classes.json'

const SERVER = process.env.REACT_APP_PY_SERVER;
const create = async (comic: any, server = SERVER) => {
  let success = true;
  const last_update = {last_update: new Date().getTime()}
  const titles = {titles: [comic.title]};
  const genres = {genres: [comic.genres]};
  const published_in = {published_in: [comic.published_in]};
  const data = {...comic, ...last_update, ...titles, ...genres, ...published_in}
  console.debug(JSON.stringify(data))
  await fetch(`${server}/comics`, {
    method: 'POST',
    body: JSON.stringify(data),
    headers: { 'Content-Type': 'application/json' },
  })
  .then((response) => response.json())
  .then((data) => {
    console.debug(data);
    if (data?.message === 'Internal Server Error') success = false;
  })
  .catch((err) => {
    console.debug(err.message);
    success = false;
  });
  return success;
};

const CreateComic = () => {
  const [isCreateComicModalOpen, setIsCreateComicModalOpen] = useState(false);
  const [comicFormData, setComicFormData] = useState<any>(null);
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

    if (await create(data)) {
      handleCloseCreateComicModal();
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
    setTimeout(() => setHideMsg(true), 1000);
    return true;
  };

  return (<>
    <button className={'button-plus'} onClick={handleOpenCreateComicModal}>
    </button>

    {(comicFormData?.title && showMsg && timerHide()) && (
      <div className={
        `msg-box ${hideMsg ? 'msg-hide' : ''} ${failMsg ? 'msg-fail' : ''}`
        }>
        <b>{db_classes?.com_type[comicFormData?.com_type]}</b> comic {' '}
        <b>{comicFormData.title}</b> {failMsg ? 'failed' : 'created'}.
      </div>
    )}

    <CreateComicModal
      isOpen={isCreateComicModalOpen}
      onSubmit={handleFormSubmit}
      onClose={handleCloseCreateComicModal}
    />
  </>);
};

export default CreateComic;