import React from 'react';
import './App.css';

const ComicCard = (props) => (
  <li key={props.comic.id} className="comicCard">
    <img className="poster" src={props.comic.cover} alt={props.comic.titles[0]}/>
    <h3>{props.comic.titles[0]}</h3>
    <span className="comicYear">Current chapter {props.comic.current_chap}</span>
    {props.comic.track ? <p className="comicYear">(Tracked) Viewed Chap: {props.comic.viewed_chap}</p> : ''}
    <p>{props.comic.author}</p>
  </li>)


class SearchDiv extends React.Component {

  constructor(props) {
    super(props);
    this.handleChange = this.handleChange.bind(this);
    this.state = {
      searchString: ''
    };
  }

  handleChange(e) {
    this.setState({searchString:e.target.value});
  }

  render() {
    let f_comics = this.props.items,
      searchString = this.state.searchString.trim().toLowerCase();

    if (searchString.length > 0) {
      // Filter the results
      f_comics = f_comics.filter((com) => {
        for (const title of com.titles) {
          if (title.toLowerCase().includes( searchString )) {
            return true;
          }
        }
        return false;
      });
    }

    return <div>
        <input  type="text"  value={this.state.searchString} 
          onChange={this.handleChange}  placeholder="Search by comic name" />
        <ul> 
          {f_comics.map((item, i) => <ComicCard comic={item} key={item.id} />)}
        </ul>
      </div>;
  }
};

const App = (props) => {
  return <>
    <SearchDiv items={ props.items } />
  </>
}

export default App;
