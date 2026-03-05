import { lazy, Suspense, useState } from 'react';
import OpMsg from './OpMsg';
import './EditComic.css';
import config from '../../../util/Config';
import db_classes from '../../../../db/db_classes.json';
import { useComicCard } from '../Card/ComicCardContext';
import Loaders from '../../Loaders';

const SERVER = config.SERVER;
const EditComicModal = lazy(() => import('../Modals/EditModal'));
const edit: (comic: Record<string, any>, server?: string) => Promise<[Record<string, any> | undefined, string]> =
  async (comic: Record<string, any>, server = SERVER) => {
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
          newData = undefined;
          msg = data.message;
        }
        else newData = data;
      })
      .catch((err) => {
        console.debug(err.message);
        newData = undefined;
        msg = err.message;
      });
    return [newData, msg];
  };

const EditComic = () => {
  const { comic, setComic, setViewedChap } = useComicCard();
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

    if (newData !== undefined) {
      setComic(newData);
      setViewedChap(newData?.viewed_chap);
      handleCloseEditComicModal();
      setFailMsg(false);
      return true;
    }
    setFailMsg(true);
    return false;
  };

  return (<>
    <button
      className='edit-button'
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

    <Suspense fallback={<Loaders selector="line-fw" />}>
      <EditComicModal
        comic={comic}
        isOpen={isEditComicModalOpen}
        onSubmit={handleFormSubmit}
        onClose={handleCloseEditComicModal}
      />
    </Suspense>
  </>);
};

export default EditComic;
