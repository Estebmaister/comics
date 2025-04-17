// All these classes come from src/db/__init__.py
import dbClasses from '../../db/db_classes.json';

const assignArrayToDict = (array: string[]) =>
  Object.assign({}, ...array.map(
    (val: string, idx: number) => ({ [idx]: val })
  )
  );

type nToString = {
  [key: number]: string
}

let Types: nToString, Statuses: nToString, Genres: nToString, Publishers: nToString;

for (let key in dbClasses) {
  switch (key) {
    case 'com_type':
      Types = assignArrayToDict(dbClasses[key]);
      break;
    case 'status':
      Statuses = assignArrayToDict(dbClasses[key]);
      break;
    case 'genres':
      Genres = assignArrayToDict(dbClasses[key]);
      break;
    case 'published_in':
      Publishers = assignArrayToDict(dbClasses[key]);
      break;

    default:
      console.error('Unexpected key in db_classes.json: ', key)
      break;
  }
}
export { Types, Statuses, Genres, Publishers };