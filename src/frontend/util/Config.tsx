const config = {
  SERVER:
    window.location.hostname !== process.env.REACT_APP_EXTERNAL_HOST
      ? `http://${window.location.hostname}:5001`
      : process.env.REACT_APP_PY_SERVER,
};

export default config;