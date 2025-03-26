import React, { useState, SetStateAction } from 'react';
import Loaders from '../../Loaders';
import './ScrapeButton.css';
import config from '../../../util/Config';

const SERVER = config.SERVER;

const scrape = async (
  setShowLoader: { (value: SetStateAction<boolean>): void; },
  server = SERVER
) => {
  let success = true;
  setShowLoader(true);
  await fetch(`${server}/scrape`, {
    method: 'GET',
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
  setShowLoader(false);
  return success;
}

const ScrapeButton = () => {
  const [showMsg, setShowMsg] = useState(false);
  const [showLoader, setShowLoader] = useState(false);
  const [hideMsg, setHideMsg] = useState(false);
  const [failMsg, setFailMsg] = useState(false);

  const handleOpenScrapeButtonModal = async () => {
    setShowMsg(false);
    setFailMsg(false);

    if (await scrape(setShowLoader)) {
      setFailMsg(false);
      setHideMsg(false);
      setShowMsg(true);
      setTimeout(() => window.location.reload(), 2000);
      return;
    }
    setHideMsg(false);
    setFailMsg(true);
    setShowMsg(true);
  };

  const timerHide = () => {
    setTimeout(() => setHideMsg(true), 1000);
    return true;
  }

  return (<>
    <button
      className={'button-scrape'}
      onClick={handleOpenScrapeButtonModal}
      disabled={showLoader}
    >
      {showLoader &&
        (<span className={'span-loader'}> <Loaders selector='battery' /> </span>)
      }
    </button>

    {(showMsg && timerHide()) && (
      <div className={
        `msg-box ${hideMsg ? 'msg-hide' : ''} ${failMsg ? 'msg-fail' : ''}`
      }>
        <b>Scrape</b> function trigger{' '}
        <b>{failMsg ? 'failed' : 'succeeded'}.</b>
      </div>
    )}

  </>);
};

export default ScrapeButton;