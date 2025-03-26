const config = {
  SERVER:
    window.location.hostname !== process.env.REACT_APP_EXTERNAL_HOST
      ? `http://${window.location.hostname}:5001`
      : process.env.REACT_APP_PY_SERVER,
};

console.log(config.SERVER);
console.log(window.location.hostname);
console.log(process.env.REACT_APP_EXTERNAL_HOST);

export default config;