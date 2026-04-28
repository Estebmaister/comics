const config = {
  SHOW_MESSAGE_TIMEOUT: 2000, // Time in ms to show message
  SERVER: // Dinamic Server URL when hosting both frontend and backend
    window.location.hostname !== import.meta.env.VITE_EXTERNAL_HOST
      ? `https://${window.location.hostname}:5001`
      : import.meta.env.VITE_PY_SERVER,
};

export default config;
