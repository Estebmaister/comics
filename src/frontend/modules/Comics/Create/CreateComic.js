import React, { useState } from 'react';
import CreateComicModal from '../CreateModal';
import './CreateComic.css';
import db_classes from '../../../../db/db_classes.json'
const SERVER = process.env.REACT_APP_PY_SERVER;

const create = async (comic, server = SERVER) => {
  let success = true;
  const last_update = {last_update: new Date().getTime()}
  const titles = {titles: [comic.title]};
  const genres = {genres: [comic.genres]};
  const published_in = {published_in: [comic.published_in]};
  const data = {...comic, ...last_update, ...titles, ...genres, ...published_in}
  console.log(JSON.stringify(data))
  comic.track = comic.track === 'true';
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
}

const CreateComic = () => {
  const [isCreateComicModalOpen, setCreateComicModalOpen] = useState(false);
  const [comicFormData, setComicFormData] = useState(null);
  const [showMsg, setShowMsg] = useState(false);
  const [hideMsg, setHideMsg] = useState(false);
  const [failMsg, setFailMsg] = useState(false);

  const handleOpenCreateComicModal = () => {
    setCreateComicModalOpen(true);
    setShowMsg(false);
    setFailMsg(false);
  };

  const handleCloseCreateComicModal = () => {
    setCreateComicModalOpen(false);
  };

  const handleFormSubmit = async (data) => {
    setComicFormData(data);

    if (await create(data)) handleCloseCreateComicModal();
    else {
      setHideMsg(false);
      setFailMsg(true);
      setShowMsg(true);
      return false;
    }
    
    setFailMsg(false);
    setHideMsg(false);
    setShowMsg(true);
    return true;
  };

  const timerHide = () => {
    setTimeout(() => setHideMsg(true), 1000);
    return true;
  }

  return (<>
    <button Class={'button-plus'} onClick={handleOpenCreateComicModal}></button>

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