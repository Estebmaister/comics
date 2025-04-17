import Loaders from '.';

const LoadMsgs = {
  network: <>
    {'Network error in attempt to connect the server'}
    <Loaders selector='lamp' />
  </>,
  server: <>{'Server internal error'} <Loaders selector='battery' /></>,
  wait: <>{'Waking up server ...'} <Loaders selector='line-fw' /></>,
  empty: (queryFilter: string) => `No comics found for title: ${queryFilter}`
}

export default LoadMsgs;