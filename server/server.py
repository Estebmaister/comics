from db.models import ComicDB, Types, Statuses, load_comics, save_comics_file
from db.comics_repo import all_comics, comics_by_title, comic_by_id, comics_by_title_no_case
from flask import Flask, jsonify, request

server = Flask(__name__)
def accepted(): return (jsonify({'message': 'Accepted'}), 202)
def not_found(msg = 'Not Found'): return (jsonify({'message': msg}), 404)
def bad_request(msg = 'Bad Request'): return (jsonify({'message': msg}), 400)

# RESTful API routes

@server.route('/')
def index():
    return '<h1>Comics Server API!</h1>'

@server.route('/comics/', methods=['GET'])
def get_comics():
    comics = all_comics()
    return jsonify({'comics': [comic.toJSON() for comic in comics]})

@server.route('/comics/<int:comic_id>/', methods=['GET'])
def get_comic(comic_id):
    comic, _ = comic_by_id(comic_id)
    if comic is None: return not_found()
    return jsonify({'comic': comic.toJSON()})

@server.route('/comics/<string:title>/', methods=['GET'])
def get_comics_by_title(title):
    comics = comics_by_title_no_case(title)
    return jsonify({'comics': [comic.toJSON() for comic in comics]})

@server.route('/comics/', methods=['POST'])
def create_comic():
    if not request.json: return bad_request()
    
    if 'titles' not in request.json: 
        return bad_request('titles is a necessary field to create a comic')
    if (type(request.json['titles']) != list or
        '' in request.json['titles']):
        return bad_request('titles should be a non-empty list of strings')
    db_comic, session = comics_by_title(request.json['titles'][0].capitalize())
    if db_comic != None: return bad_request('Title is already in the database')

    if ('description' in request.json and 
        type(request.json['description']) is not str):
        return bad_request('description type different from string')
    if 'track' in request.json and type(request.json['track']) is not bool:
        return bad_request('track type different from boolean')
    if 'viewed_chap' in request.json: int(request.json['track'])

    comic = ComicDB(
        id     = None,
        titles = None,
        current_chap = request.json.get('current_chap', 0),
        cover        = request.json.get('cover', ''),
        com_type     = int(request.json.get('com_type', 0)),
        status       = int(request.json.get('status', 0)),
        description  = request.json.get('description', ''),
        author       = request.json.get('author', ''),
        track        = int(request.json.get('track', 0)),
        viewed_chap  = int(request.json.get('viewed_chap', 0))
    )
    comic.set_titles( request.json.get('titles', ['']) )
    comic.set_published_in( request.json.get('published_in', [0]) )
    comic.set_genres( request.json.get('genres', [0]) )

    session.add(comic)
    session.commit()
    load_comics.append(comic.toJSON())
    save_comics_file(load_comics)
    return jsonify({'comic': comic.toJSON()}), 201

@server.route('/comics/<int:comic_id>/', methods=['PUT'])
def update_comic(comic_id):
    comic, session = comic_by_id(comic_id)
    if comic is None: return not_found()
    json_comic = [comic for comic in load_comics if comic_id == comic["id"]][0]

    if not request.json: return bad_request('Impossible to read json body')

    if 'titles' in request.json and (type(request.json['titles']) != list or
        '' in request.json['titles']):
        return bad_request('titles should be a non-empty list')
    if 'author' in request.json and type(request.json['author']) is not str:
        return bad_request('author type different from string')
    if ('cover' in request.json 
        and (type(request.json['cover']) is not str 
        or "http" not in request.json['cover'])):
            return bad_request('cover should be a correct http link')
    if ('description' in request.json and 
        type(request.json['description']) is not str):
        return bad_request('description type different from string')
    if 'track' in request.json and type(request.json['track']) is not bool:
        return bad_request('track type different from boolean')
    if 'viewed_chap' in request.json: int(request.json['track'])
    if 'com_type'    in request.json: int(request.json['com_type'])
    if 'status'      in request.json: int(request.json['status'])

    # genres
    # published_in

    titles = request.json.get('titles')
    if titles != None:
        comic.set_titles(titles)
        json_comic["titles"] = titles
    
    comic.author = request.json.get('author', comic.author)
    comic.cover =  request.json.get('cover', comic.cover)
    comic.description = request.json.get('description', comic.description)
    comic.track       = int(request.json.get('track', comic.track))
    comic.viewed_chap = int(request.json.get('viewed_chap', comic.viewed_chap))
    comic.com_type    = int(request.json.get('com_type', comic.com_type))
    comic.status      = int(request.json.get('status', comic.status))

    json_comic["author"] = comic.author
    json_comic["cover"]  = comic.cover
    json_comic["description"] = comic.description
    json_comic["track"]       = bool(comic.track)
    json_comic["viewed_chap"] = comic.viewed_chap
    json_comic["com_type"]    = Types(comic.com_type)
    json_comic["status"]      = Statuses(comic.status)

    session.commit()
    save_comics_file(load_comics)
    return jsonify({'comic': comic.toJSON()})

@server.route('/comics/<int:comic_id>/', methods=['DELETE'])
def delete_comic(comic_id):
    comic, session = comic_by_id(comic_id)
    if comic is None: return not_found()
    dj_comic = [com for com in load_comics if comic.id == com["id"]][0]
    session.delete(comic)
    session.commit()
    load_comics.remove(dj_comic)
    save_comics_file(load_comics)
    return accepted()

@server.route('/comics/<int:comic_id>/<int:comic_merging_id>/', 
    methods=['PUT', 'PATCH'])
def merge_comics(comic_id, comic_merging_id):
    comic, session = comic_by_id(comic_id)
    if comic is None: return not_found(f'id {comic_id} not found')
    d_comic = session.query(ComicDB).get(comic_merging_id)
    if d_comic is None: return not_found(f'id {comic_merging_id} not found')
    if comic.com_type != d_comic.com_type:
        return bad_request(f'comics to merge should be of the same type')
    json_comic = [com for com in load_comics if comic.id == com["id"]][0]
    dj_comic = [com for com in load_comics if d_comic.id == com["id"]][0]

    titles = list(set(comic.get_titles() + d_comic.get_titles()))
    comic.set_titles(titles)
    genres = list(set(comic.get_genres() + d_comic.get_genres()))
    comic.set_genres(genres)
    publishers = list(set(comic.get_published_in() +d_comic.get_published_in()))
    comic.set_published_in(publishers)
    if comic.current_chap < d_comic.current_chap:
        comic.current_chap = d_comic.current_chap
    
    json_comic["titles"] = titles
    json_comic["genres"] = genres
    json_comic["published_in"] = publishers
    json_comic["com_type"]    = Types(comic.com_type)

    session.delete(d_comic)
    session.commit()
    load_comics.remove(dj_comic)
    save_comics_file(load_comics)
    return accepted()

# API Error handling

@server.errorhandler(404)
def handle_bad_request(e):
    return bad_request('Check the URL used')

@server.errorhandler(ValueError)        
def value_error(e):
    server.logger.error(e)
    return bad_request('Bad request, check the data format')
@server.errorhandler(ZeroDivisionError)
def zero_division_error(e):
    server.logger.error(e)
    return 'Internal division by zero, please report this error', 500      
@server.errorhandler(IndexError)
def index_error(e):
    server.logger.error(e)
    return 'Internal bad index access, please report this error', 500
@server.errorhandler(TypeError)
def type_error(e):
    server.logger.error(e)
    return 'Internal bad type creation, please report this error', 500
