import { useState, SetStateAction } from 'react';
import EditComicModal from '../Modals/EditModal';
import './EditComic.css';
import db_classes from '../../../../db/db_classes.json'

const SERVER = process.env.REACT_APP_PY_SERVER;
const edit = async (comic: any, server = SERVER) => {
  let newData;
  comic.last_update = new Date().getTime();
  const data = { ...comic };
  console.debug(JSON.stringify(data))
  await fetch(`${server}/comics/${comic.id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
    headers: { 'Content-Type': 'application/json' },
  })
    .then((response) => response.json())
    .then((data) => {
      console.debug(data);
      if (data?.message !== undefined) newData = null;
      else newData = data;
    })
    .catch((err) => {
      console.debug(err.message);
      newData = null;
    });
  return newData;
};

const EditComic = (props: {
  comic: any,
  setComic: { (value: SetStateAction<any>): void; },
  setViewed: { (value: SetStateAction<boolean>): void; },
}) => {
  const { comic, setComic, setViewed } = props;
  const [isEditComicModalOpen, setIsEditComicModalOpen] = useState(false);
  const [comicFormData, setComicFormData] = useState(comic);
  const [showMsg, setShowMsg] = useState(false);
  const [hideMsg, setHideMsg] = useState(false);
  const [failMsg, setFailMsg] = useState(false);

  const handleOpenEditComicModal = () => {
    setIsEditComicModalOpen(true);
    setShowMsg(false);
    setFailMsg(false);
  };

  const handleCloseEditComicModal = () => {
    setIsEditComicModalOpen(false);
  };

  const handleFormSubmit = async (data: {}) => {
    setComicFormData(data);

    const newData: any = await edit(data);
    if (newData != null) {
      setComic(newData);
      setComicFormData(newData);
      setViewed(newData?.viewed_chap);
      handleCloseEditComicModal();
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
    <button
      className={'edit-button basic-button reverse-button'}
      onClick={handleOpenEditComicModal}
    >
      EDIT
    </button>

    {(showMsg && timerHide()) && (
      <div className={
        `msg-box ${hideMsg ? 'msg-hide' : ''} ${failMsg ? 'msg-fail' : ''}`
      }>
        <b>{db_classes?.com_type[comicFormData?.com_type]}</b> comic {' '}
        <b>{comicFormData.titles}</b> {failMsg ? 'failed' : 'created'}.
      </div>
    )}

    <EditComicModal
      isOpen={isEditComicModalOpen}
      onSubmit={handleFormSubmit}
      onClose={handleCloseEditComicModal}
      comic={comic}
    />
  </>);
};

export default EditComic;