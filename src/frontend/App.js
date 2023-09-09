import React from 'react';
import './App.css';
import comics from '../db/comics.json';
import { ComicCard } from './ComicCard';

comics.sort((a,b) => b.last_update - a.last_update)
export const COMIC_SEARCH_PLACEHOLDER = "Search by comic name";

const filter_comics = (comics, from, limit, filter_word, tracked_only) => 
  comics.filter((com) => {
    for (const title of com.titles) {
      if (tracked_only && !com.track) {
        return false;
      }
      if (title.toLowerCase().includes( filter_word.toLowerCase() )) {
        return true;
      }
    }
    return false;
  }).slice( from, from + limit );

class SearchDiv extends React.Component {
  constructor(props) {
    super(props);
    this.handleInputChange = this.handleInputChange.bind(this);
    this.handlePagination = this.handlePagination.bind(this);
    this.state = {
      username: '',
      // A string with filtering words.
      search_string: localStorage.getItem('search_string') || '',
      // A slice with current 8 filtered comics.
      f_comics: filter_comics(
        comics, parseInt(localStorage.getItem('from')) || 0, 8, 
        localStorage.getItem('search_string') || '', 
        JSON.parse(localStorage.getItem('tracked_only'))
      ),
      // A flag for the tracked comics
      tracked_only: JSON.parse(localStorage.getItem('tracked_only')),
      // Pagination
      from: parseInt(localStorage.getItem('from')) || 0,
      limit: 8
    };
  }

  handleTrackedOnly() {
    localStorage.setItem('tracked_only', !this.state.tracked_only);
    this.setState((state, _props) => ({
      tracked_only: !state.tracked_only,
    }), () => this.handleInputChange());
  }

  handleInputChange(e) {
    let filter_word = e?.target?.value === undefined ? 
      this.state.search_string : e.target.value.trim();
    localStorage.setItem('search_string', filter_word);

    let filtered_comics;
    if (filter_word !== '' || this.state.tracked_only === true) {
      filtered_comics = filter_comics(comics, 
        this.state.from, this.state.from + this.state.limit, 
        filter_word, this.state.tracked_only);
    } else {
      filtered_comics = comics.slice(
        this.state.from, this.state.from + this.state.limit
        );
    }

    this.setState((_state, _props) => ({
      search_string: filter_word,
      f_comics: filtered_comics
    }), () => console.log(this.state));
  }

  handlePagination(direction) {
    let moveFrom = 0
    if (direction === 'next') {
      moveFrom = +this.state.limit;
    } else if (direction === 'prev') {
      moveFrom = -this.state.limit;
    } else {
      console.log("Pagination called without valid argument: ", direction);
      return;
    }

    localStorage.setItem('from', this.state.from + moveFrom);
    this.setState((state, _props) => ({
      from: state.from + moveFrom,
    }), () => this.handleInputChange());
  }

  render() {
    return <div>
      <div className="nav-bar">
        <button className="basic-button search-button" 
          onClick={() => this.handleTrackedOnly()} > Tracked
        </button>

        <input className="search-box" type="text" 
          placeholder={COMIC_SEARCH_PLACEHOLDER}
          value={this.state.search_string} onChange={this.handleInputChange}
        />

        <div className='pagination-buttons'>
          {this.state.from === 0 ? '' : 
            <button className='basic-button search-button untrack-button' 
              onClick={() => this.handlePagination('prev')}>Prev</button>
          }
          <button className='basic-button search-button' 
            onClick={() => this.handlePagination('next')} >
            Next
          </button>
        </div>
      </div>

      <ul className='comic-list'> {
        this.state.f_comics.map( (item, _i) => 
          <ComicCard comic={item} key={item.id} />
        )
      } </ul>
    
    </div>;
  }
};

export const App = () => {
  return <>
    <SearchDiv/>
  </>
}