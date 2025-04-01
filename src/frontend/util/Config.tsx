const config = {
  SHOW_MESSAGE_TIMEOUT: 2000, // Time in ms to show message
  SERVER: // Dinamic Server URL when hosting both frontend and backend
    window.location.hostname !== process.env.REACT_APP_EXTERNAL_HOST
      ? `http://${window.location.hostname}:5001`
      : process.env.REACT_APP_PY_SERVER,
};

export default config;