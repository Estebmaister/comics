import React, { useState } from 'react';
import CreateComicModal from '../CreateModal';
import './CreateComic.css';
import db_classes from '../../../../db/db_classes.json'


const CreateComic = () => {
  const [isCreateComicModalOpen, setCreateComicModalOpen] = useState(false);
  const [comicFormData, setComicFormData] = useState(null);
  const [showMessage, setShowMessage] = useState(false);
  const [hideMessage, setHideMessage] = useState(false);

  const handleOpenCreateComicModal = () => {
    setCreateComicModalOpen(true);
    setShowMessage(false);
  };

  const handleCloseCreateComicModal = () => {
    setCreateComicModalOpen(false);
  };

  const handleFormSubmit = (data) => {
    setComicFormData(data);
    handleCloseCreateComicModal();
    setHideMessage(false);
    setShowMessage(true);
  };

  const timerHide = () => {
    setTimeout(() => setHideMessage(true), 1000);
    return true;
  }

  return (<>
    <button Class={'button-plus'} onClick={handleOpenCreateComicModal}></button>

    {(comicFormData?.title && showMessage && timerHide()) && (
      <div className={`msg-box ${hideMessage ? 'msg-hide' : ''}`}>
        <b>{db_classes?.com_type[comicFormData?.com_type]}</b> comic {' '}
        <b>{comicFormData.title}</b> created.
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