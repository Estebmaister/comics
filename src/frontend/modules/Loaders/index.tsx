import './Loaders.css';

const loaderSwitch = (selector: string) => {
  switch (selector) {
    case 'lamp':
      return <span className="lamp"></span>;
    case 'line-fw':
      return <span className="line-fw"></span>;
    case 'battery':
    default:
      return <span className="battery"></span>;
  }
}

const Loaders = ({selector = 'battery'}) => {
  return <div className="div-load">
    {loaderSwitch(selector)}
  </div>
}

export default Loaders