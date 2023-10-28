import './Loaders.css';

const loaderSwitch = (selector) => {
  switch (selector) {
    case 'lamp':
      return <span class="lamp"></span>;
    case 'line-fw':
      return <span class="line-fw"></span>;
    case 'battery':
    default:
      return <span class="battery"></span>;
  }
}

const Loaders = ({selector = 'battery'}) => {
  return <div class="div-load">
    {loaderSwitch(selector)}
  </div>
}

export default Loaders