import React from 'react';
import './App.css';

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
  
  console.log(JSON.stringify({
    "track": !tracked
  }))
  fetch(server + '/comics/' + id, {
    method: 'PUT',
    body: JSON.stringify({
      track: !tracked
    }),
    headers: {
      'Content-Type': 'application/json',
    },
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
