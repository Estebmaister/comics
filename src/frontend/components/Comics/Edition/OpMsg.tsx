import React from 'react';
import './OperationMsg.css';
import config from '../../../util/Config';

const SHOW_MESSAGE_TIMEOUT = config.SHOW_MESSAGE_TIMEOUT;
const timerHide = (setHideMsg: (value: boolean) => void) => {
  setTimeout(() => setHideMsg(true), SHOW_MESSAGE_TIMEOUT);
  return true;
};

const OpMsg: React.FC<OpMsgProps> = ({ comicType, title, msg, operation, showMsg, hideMsg, failMsg, setHideMsg }) => {
  return (<>
    {(showMsg && timerHide(setHideMsg)) && (
      <div
        className={`msg-box ${hideMsg ? 'msg-hide' : ''} ${failMsg ? 'msg-fail' : ''}`}
      >
        {/* Manga comic Murim Login operation (failed/succeed). reason...*/}
        <b>{comicType}</b> comic {' '}
        <b>{title}</b> {operation} {failMsg ? 'failed' : 'succeeded'}. {msg}
      </div>
    )}
  </>);
};

interface OpMsgProps {
  comicType?: string;
  title?: string;
  msg?: string;
  operation: string;
  showMsg: boolean;
  hideMsg: boolean;
  failMsg: boolean;
  setHideMsg: (value: boolean) => void;
}

export default OpMsg;