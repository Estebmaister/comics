import React from 'react';
import ReactDOM from 'react-dom/client';
import './frontend/index.css';
import App from './frontend/App';
import reportWebVitals from './frontend/reportWebVitals';
import comics from './db/comics.json';

comics.sort((a,b) => a.last_update - b.last_update)

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <React.StrictMode>
    <App items={comics}/>
  </React.StrictMode>
);

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
