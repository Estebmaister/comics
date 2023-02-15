import React from 'react';
import './App.css';
import comics from '../db/comics.json';

comics.sort((a,b) => a.last_update - b.last_update)

const Types = {
    0: 'Unknown',
    1: 'Manga',
    2: 'Manhua',
    3: 'Manhwa', 
    4: 'Novel'
}

const Statuses = {
    0: 'Unknown',  
    1: 'Completed',
    2: 'OnAir',    
    3: 'Break',    
    4: 'Dropped',  
}

const Genres = {
    0: 'Unknown',      
    1: 'Action',       
    2: 'Adventure',    
    3: 'Fantasy',      
    4: 'Overpowered',  
    5: 'Comedy',       
    6: 'Drama',        
    7: 'SchoolLife',   
    8: 'System',       
    9: 'Supernatural', 
    10:'MartialArts',  
    11:'Romance',      
    12:'Shounen',      
    13:'Reincarnation',
}

const Publishers = {
    0: 'Unknown',      
    1: 'Asura',        
    2: 'ReaperScans',  
    3: 'ManhuaPlus',   
    4: 'FlameScans',   
    5: 'LuminousScans',
    6: 'ResetScans',   
    7: 'IsekaiScan',   
    8: 'RealmScans',   
}

const publishers_handler = (publishers) => {
  const return_array = [];
  publishers.forEach( element => 
    return_array.push(Publishers[+element]) )
  return return_array.join(', ');
}

const genres_handler = (genres) => {
  const return_array = [];
  genres.forEach( element => 
    return_array.push(Genres[+element]) )
  return return_array.join(', ');
}

const server = "http://localhost:5000"

const track = (tracked, id) => {
  fetch(`${server}/comics/${id}`, {
    method: 'PUT',
    body: JSON.stringify({ track: !tracked }),
    headers: { 'Content-Type': 'application/json' },
  })
    .then((response) => response.json())
    .then((data) => {
        console.log(data);
    })
    .catch((err) => {
        console.log(err.message);
    });
}

const ComicCard = (props) => (
  <li key={props.comic.id} className="comic-card">
    <img className="poster" src={props.comic.cover} alt={props.comic.titles[0]}/>
    <h3>{props.comic.titles[0]}</h3>
    <span className="comic-year">Current Chap {props.comic.current_chap}</span>
    {props.comic.track ? 
      <p className="comic-year">Viewed Chap {props.comic.viewed_chap}</p> 
      : ''}
    <p>{props.comic.author}</p>
    <p>{Types[props.comic.com_type]}</p>
    <p>Status: {Statuses[props.comic.status]}</p>
    <p>Genres: {genres_handler(props.comic.genres)}</p>
    <p>{publishers_handler(props.comic.published_in)}</p>
    <button className="track-button" 
      onClick={() => track(props.comic.track, props.comic.id)}>
      {props.comic.track ? "Untrack":"Track"}
    </button>
  </li>)


class SearchDiv extends React.Component {

  constructor(props) {
    super(props);
    this.handleChange = this.handleChange.bind(this);
    this.state = {
      searchString: '',
      f_comics: comics.slice(0, 8),
      from: 0,
      limit: 8
    };
  }

  handleChange(e) {
    let filtered_comics = comics.slice(this.state.from, this.state.limit)
    if (e.target.value.trim().length > 0) {
      filtered_comics = comics.filter((com) => {
        for (const title of com.titles) {
          if (title.toLowerCase().includes( e.target.value.trim().toLowerCase() )) {
            return true;
          }
        }
        return false;
      });
    }

    this.setState((state, _props) => ({
      searchString: e.target.value,
      f_comics: filtered_comics.slice(state.from, state.limit)
    }));
  }

  render() {

    return <div>
      <input type="text" value={this.state.searchString} 
        onChange={this.handleChange}  placeholder="Search by comic name" />
      <ul>
        {this.state.f_comics.map((item, _i) => <ComicCard comic={item} key={item.id} />)}
      </ul>
      {this.state.from === 0 ? '' : <button>Prev</button>}
      <button>Next</button>
    </div>;
  }
};

const App = () => {
  return <>
    <SearchDiv/>
  </>
}

export default App;
