import { useState, SetStateAction } from 'react';
import EditComicModal from '../Modals/EditModal';
import OpMsg from './OpMsg';
import './EditComic.css';
import config from '../../../util/Config';
import db_classes from '../../../../db/db_classes.json';

const SERVER = config.SERVER;
const edit: (comic: any, server?: string) => Promise<[any | null, string]> =
  async (comic: any, server = SERVER) => {
    let newData;
    let msg = '';
    comic.last_update = new Date().getTime();
    const data = { ...comic };
    delete data.deleted;
    delete data.track;
    console.debug(JSON.stringify(data))
    await fetch(`${server}/comics/${comic.id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
      headers: { 'Content-Type': 'application/json' },
    })
      .then((response) => response.json())
      .then((data) => {
        console.debug(data);
        if (data?.message !== undefined) {
          newData = null;
          msg = data.message;
        }
        else newData = data;
      })
      .catch((err) => {
        console.debug(err.message);
        newData = null;
        msg = err.message;
      });
    return [newData, msg];
  };

const EditComic = (props: {
  comic: any,
  setComic: { (value: SetStateAction<any>): void; },
  setViewed: { (value: SetStateAction<boolean>): void; },
}) => {
  const { comic, setComic, setViewed } = props;
  const [isEditComicModalOpen, setIsEditComicModalOpen] = useState(false);
  const [msg, setMsg] = useState('');
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
    const [newData, resultMsg] = await edit(data);
    setMsg(resultMsg);
    setHideMsg(false);
    setShowMsg(true);

    if (newData != null) {
      setComic(newData);
      setViewed(newData?.viewed_chap);
      handleCloseEditComicModal();
      setFailMsg(false);
      return true;
    }
    setFailMsg(true);
    return false;
  };

  return (<>
    <button
      className={'edit-button basic-button reverse-button'}
      onClick={handleOpenEditComicModal}
    >
      EDIT
    </button>


    <OpMsg
      msg={msg}
      operation="update"
      showMsg={showMsg}
      hideMsg={hideMsg}
      failMsg={failMsg}
      setHideMsg={setHideMsg}
      comicType={db_classes?.com_type[comic?.com_type]}
      title={comic?.titles}
    />

    <EditComicModal
      comic={comic}
      isOpen={isEditComicModalOpen}
      onSubmit={handleFormSubmit}
      onClose={handleCloseEditComicModal}
    />
  </>);
};

export default EditComic;