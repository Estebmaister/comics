import { Types, Statuses, Genres, Publishers } from './Classes';
import BrokenImage from './404.jpg'
import styles from'./ComicCard.module.css'
const server = 'http://localhost:5000'

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

const checkout = (curr_chap, id) => {
  fetch(`${server}/comics/${id}`, {
    method: 'PUT',
    body: JSON.stringify({ viewed_chap: curr_chap }),
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

export const ComicCard = (props) => (
    <li key={props.comic.id} className={styles.comicCard}>
      <img className={styles.poster} src={props.comic.cover} alt={props.comic.titles[0]}
        onError={(event) => event.currentTarget.src = BrokenImage}
      />
      <h3 className={styles.comicTitle}>{props.comic.titles[0]}</h3>
      
      <p className={styles.comicChapter + ' text'}>Current Chapter 
        {props.comic.track && props.comic.current_chap!==props.comic.viewed_chap ?
        (<span className={styles.currentChapter}> {props.comic.viewed_chap+'/ '}</span>) 
        : ' '}
        {props.comic.current_chap}
      </p>
      
      <p className='text'> {props.comic.author} </p>
      <p className='text'>
        Status: {Statuses[props.comic.status]}
      </p>
      <p className='text'>
        Type: {Types[props.comic.com_type]}
      </p>
      <p className='text'> 
        Genres: {genres_handler(props.comic.genres)}
      </p>
      <p className='text'>
        Publishers: {publishers_handler(props.comic.published_in)}
      </p>

      {props.comic.track && props.comic.current_chap !== props.comic.viewed_chap ? 
        <button className='basic-button track-button check-button' 
          onClick={() => checkout(props.comic.current_chap, props.comic.id)}>
          Checkout
        </button> : ''
      }
      <button className={'basic-button track-button' + 
        (props.comic.track ? ' untrack-button' : '')} 
        onClick={() => track(props.comic.track, props.comic.id)}>
        {props.comic.track ? 'Untrack':'Track'}
      </button>
    </li>)