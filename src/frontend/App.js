import React from 'react';
import './App.css';
import comics from '../db/comics.json';
import { ComicCard } from './ComicCard';

comics.sort((a,b) => b.last_update - a.last_update)
export const COMIC_SEARCH_PLACEHOLDER = "Search by comic name";

const filter_comics = (comics, filter_word, tracked_only) => 
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
  });

class SearchDiv extends React.Component {
  constructor(props) {
    super(props);
    this.handleInputChange = this.handleInputChange.bind(this);
    this.handlePagination = this.handlePagination.bind(this);
    this.handleTrackedOnly = this.handleTrackedOnly.bind(this);

    const FROM = parseInt(localStorage.getItem('from')) || 0;
    const LIMIT = 8;
    const SEARCH_STRING = localStorage.getItem('search_string') || '';
    const TRACKED_ONLY = JSON.parse(localStorage.getItem('tracked_only'));
    const FILTERED_COMICS = filter_comics(comics, SEARCH_STRING, TRACKED_ONLY)
    const TOTAL = FILTERED_COMICS.length;
    
    this.state = {
      username: '',
      // A string with filtering words.
      search_string: SEARCH_STRING,
      // A slice with current 8 filtered comics.
      f_comics: FILTERED_COMICS.slice(FROM, FROM + LIMIT),
      // A flag for the tracked comics
      tracked_only: TRACKED_ONLY,
      // Pagination
      from: FROM,
      limit: LIMIT,
      total: TOTAL,
      totalPages: Math.ceil(TOTAL/LIMIT),
      currentPage: Math.ceil(FROM/LIMIT +1)
    };
  }

  handleTrackedOnly() {
    localStorage.setItem('tracked_only', !this.state.tracked_only);
    this.setState((state, _props) => ({
      tracked_only: !state.tracked_only,
      from: 0
    }), () => this.handleInputChange());
  }

  handleInputChange(e) {
    let filter_word = e?.target?.value === undefined ? 
      this.state.search_string : e.target.value.trim();
    
    let reset_pag = false;
    if (filter_word !== this.state.search_string && 
      this.state.from !== 0) reset_pag = true;
    
    localStorage.setItem('search_string', filter_word);
    
    const FROM = reset_pag ? 0 : this.state.from;
    const LIMIT = this.state.limit;
    let filtered_comics;
    let new_total;
    
    if (filter_word !== '' || this.state.tracked_only === true) {
      filtered_comics = filter_comics(
        comics, filter_word, this.state.tracked_only
      );
      new_total = filtered_comics.length;
      filtered_comics = filtered_comics.slice(FROM, FROM + LIMIT);
    } else {
      filtered_comics = comics.slice(FROM, FROM + LIMIT);
      new_total = comics.length;
    }

    this.setState((_state, _props) => ({
      search_string: filter_word,
      f_comics: filtered_comics,
      from: FROM,
      total: new_total,
      totalPages: Math.ceil(new_total/LIMIT),
      currentPage: Math.ceil(FROM/LIMIT +1)
    }), () => console.debug(this.state));
  }

  handlePagination(direction) {
    /* TODO: Final page conditions from state */
    const LAST_FROM = this.state.limit*(this.state.totalPages-1)
    let moveFrom = 0
    if (direction === 'next') {
      moveFrom = +this.state.limit;
    } else if (direction === 'prev') {
      moveFrom = -this.state.limit;
    } else if (direction === 'first') {
      moveFrom = -this.state.from;
    } else if (direction === 'last') {
      moveFrom = -this.state.from + LAST_FROM;
    }  else {
      console.error("Pagination called without valid argument: ", direction);
      return;
    }

    // Border cases, before first page, after last page
    if (this.state.from + moveFrom < 0) {
      moveFrom = -this.state.from;
    } else if (this.state.from + moveFrom > LAST_FROM) {
      moveFrom = -this.state.from + LAST_FROM;
    }
    localStorage.setItem('from', this.state.from + moveFrom);
    this.setState((state, _props) => ({
      from: state.from + moveFrom,
    }), () => this.handleInputChange());
  }

  render() {
    return <div>
      <div className='nav-bar'>
        <button className={'basic-button all-track-button' + 
          (this.state.tracked_only ? ' reverse-button' : '')} 
          onClick={() => this.handleTrackedOnly()} > 
          {this.state.tracked_only ? 'All >' : 'Tracked <'} ({this.state.total})
        </button>
        {/* TODO: New 'Unchecked button' and filter */}

        <input className='search-box' type="text" 
          placeholder={COMIC_SEARCH_PLACEHOLDER}
          value={this.state.search_string} onChange={this.handleInputChange}
        />

        <div className='div-pagination-buttons'>
          <button className={'basic-button bar-button reverse-button' + 
              (this.state.from <= 0 ? ' disabled-button' : '')}
            disabled={this.state.from <= 0}
            onClick={() => this.handlePagination('first')}>
              First
          </button>
          <button className={'basic-button bar-button reverse-button' + 
              (this.state.from <= 0 ? ' disabled-button' : '')} 
            disabled={this.state.from <= 0}
            onClick={() => this.handlePagination('prev')}>
              Prev
          </button>
          <button className={'pag-button'}>{this.state.currentPage}</button>
          <button className={'basic-button bar-button' +
              (this.state.from >= this.state.limit*(this.state.totalPages-1)
                ? ' disabled-button' : '')}
            disabled={
              this.state.from >= this.state.limit*(this.state.totalPages-1) ? 
                ' disabled-button' : 
                ''
            }
            onClick={() => this.handlePagination('next')} >
              Next
          </button>
          <button className={'basic-button bar-button' +
              (this.state.from >= this.state.limit*(this.state.totalPages-1)
              ? ' disabled-button' : '')}
            disabled={
              this.state.from >= this.state.limit*(this.state.totalPages-1) ? 
              ' disabled-button' : 
              ''
            }
            onClick={() => this.handlePagination('last')} >
              Last ({this.state.totalPages})
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