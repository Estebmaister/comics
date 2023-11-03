// All these classes come from src/db/__init__.py
import db_classes from '../../db/db_classes.json';

const assignArrayToDict = (array) => 
  Object.assign({}, ...array.map(
    (val, idx) => ({[idx]: val})
  ));

let Types, Statuses, Genres, Publishers;
for (let key in db_classes) {
  switch (key) {
    case 'com_type':
      Types = assignArrayToDict(db_classes[key]);
      break;
    case 'status':
      Statuses = assignArrayToDict(db_classes[key]);
      break;
    case 'genres':
      Genres = assignArrayToDict(db_classes[key]);
      break;
    case 'published_in':
      Publishers = assignArrayToDict(db_classes[key]);
      break;

    default:
      console.error('Unexpected key in db_classes.json: ', key)
      break;
  }
}
export {Types, Statuses, Genres, Publishers};