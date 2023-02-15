# server/__init__.py

from db import swagger_model, load_comics, save_comics_file
from db import ComicDB, Types, Statuses
from db.repo import all_comics, comics_by_title, comic_by_id
from db.repo import comics_by_title_no_case, merge_comics
from helpers.server import put_body_parser
from flask import Flask, jsonify, request
from flask_cors import CORS
from flask_restx import Api, Resource
from werkzeug.middleware.proxy_fix import ProxyFix

server = Flask(__name__)
# CORS(server)
server.config["RESTX_MASK_SWAGGER"]=False
server.wsgi_app = ProxyFix(server.wsgi_app)
api = Api(server, version='1.0', title='ComicMVC API',
    description='A Comic API capable enough to provide all CRUD ops and more',
)
ns = api.namespace('comics', description='Comic operations')
comic_rest_model = api.model('Comic', swagger_model)

# RESTful API routes

@ns.route('/')
@ns.response(400, 'Bad request')
class ComicList(Resource):
    '''Shows a list of all comics, and let you POST to add new comics'''

    @ns.doc('list_comics', params={
	'from': {'default': '0', 'description': 'Offset for query', 'type': 'int'},
	'limit': {'default': '20', 'description': 'Number of comics', 'type': 'int'}
    })
    @ns.marshal_list_with(comic_rest_model)
    def get(self):
        '''List all comics with pagination'''
        offset = request.args.get("from", 0)
        limit = request.args.get("limit", 20)
        try:
            int(offset), int(limit)
        except ValueError:
            ns.logger.info('Pagination parameters type different from int. '+
                f'[offset: {offset}, limit: {limit}]')
            api.abort(400, 'Pagination parameters type different from int')
        return [comic.toJSON() for comic in all_comics(offset, limit)]

    @ns.doc('create_comic')
    @ns.expect(comic_rest_model)
    @ns.marshal_with(comic_rest_model, code=201)
    def post(self):
        '''Create a new comic'''
        if not request.json:  api.abort(400, 'Body payload is necessary')
        if 'titles' not in request.json: 
            api.abort(400, 'titles is a necessary field to create a comic')
        if (type(request.json['titles']) != list or
            '' in request.json['titles']):
            api.abort(400, 'titles should be a non-empty list of strings')
        first_title = request.json['titles'][0].capitalize()
        db_comic, session = comics_by_title(first_title)
        if db_comic != None:
            for comic in db_comic:
                if first_title in comic.get_titles():
                    api.abort(400, 'Comic is already in the database')

        if ('description' in request.json and 
            type(request.json['description']) is not str):
            api.abort(400, 'description type different from string')
        if 'track' in request.json and type(request.json['track']) is not bool:
            api.abort(400, 'track type different from boolean')
        if 'viewed_chap' in request.json: int(request.json['track'])

        comic = ComicDB(
            id     = request.json.get('id', None),
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
        comic.set_titles(       request.json.get('titles', ['']) )
        comic.set_published_in( request.json.get('published_in', [0]) )
        comic.set_genres(       request.json.get('genres', [0]) )

        session.add(comic)
        session.commit()
        load_comics.append(comic.toJSON())
        save_comics_file(load_comics)
        return comic.toJSON()

COMIC_NOT_FOUND = 'Comic {} not found'
@ns.route('/<int:id>','/<int:id>/')
@ns.response(404, COMIC_NOT_FOUND)
@ns.param('id', 'The comic identifier')
class ComicID(Resource):
    '''Shows a single comic item and lets you delete or update by ID'''

    @ns.doc('get_comic')
    @ns.marshal_with(comic_rest_model)
    def get(self, id):
        '''Fetch a comic by ID'''
        comic, _ = comic_by_id(id)
        if comic is None: api.abort(404, COMIC_NOT_FOUND.format(id))
        return comic.toJSON()
    
    @ns.doc('delete_comic')
    @ns.response(202, 'Comic deleted')
    def delete(self, id):
        '''Delete a comic given its identifier'''
        comic, session = comic_by_id(id)
        if comic is None:  api.abort(404, COMIC_NOT_FOUND.format(id))
        dj_comic = [com for com in load_comics if comic.id == com["id"]][0]
        session.delete(comic)
        session.commit()
        load_comics.remove(dj_comic)
        save_comics_file(load_comics)
        return 202

    @ns.doc('update_comic')
    @ns.expect(comic_rest_model)
    @ns.marshal_with(comic_rest_model)
    def put(self, id):
        '''Update a comic given its identifier'''
        if not request.json:  api.abort(400, 'Body payload is necessary')
        err_reading_body: str = put_body_parser(request.json)
        if err_reading_body != '': api.abort(400, err_reading_body)

        comic, session = comic_by_id(id)
        if comic is None:  api.abort(404, COMIC_NOT_FOUND.format(id))
        json_comic = [comic for comic in load_comics if id == comic["id"]][0]

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
        genres = request.json.get('genres')
        if genres != None:
            genres = list(set([int(g) for g in request.json.get('genres', 0)]))
            comic.set_genres(genres)
            json_comic["genres"] = genres
        publishers = request.json.get('published_in')
        if publishers != None:
            publishers = list(set([int(g) for g in request.json.get('published_in', 0)]))
            comic.set_published_in(publishers)
            json_comic["published_in"] = publishers

        json_comic["author"] = comic.author
        json_comic["cover"]  = comic.cover
        json_comic["description"] = comic.description
        json_comic["track"]       = bool(comic.track)
        json_comic["viewed_chap"] = comic.viewed_chap
        json_comic["com_type"]    = Types(comic.com_type)
        json_comic["status"]      = Statuses(comic.status)

        session.commit()
        save_comics_file(load_comics)
        return comic.toJSON()

@ns.route('/<string:title>','/<string:title>/')
@ns.response(400, 'Empty title cannot be resolved')
@ns.param('title', 'The name of the comic')
class ComicTitle(Resource):
    '''List comics by title'''

    @ns.doc('get_comic_by_title')
    @ns.marshal_list_with(comic_rest_model)
    def get(self, title):
        '''Fetch a list of comics by title'''
        title = title.strip()
        if title == '': api.abort(400, 'Empty title cannot be resolved')
        return [comic.toJSON() for comic in comics_by_title_no_case(title)]

@ns.route('/<int:base_id>/<int:merging_id>','/<int:base_id>/<int:merging_id>/')
@ns.response(404, COMIC_NOT_FOUND)
@ns.response(400, 'Comics should be of the same type')
class ComicMerge(Resource):
    '''Merge comics by id'''

    @ns.doc('merge_comics')
    @ns.marshal_list_with(comic_rest_model)
    def patch(self, base_id, merging_id):
        '''Merge two comics by their respective id'''
        comic, error = merge_comics(base_id, merging_id)
        if error != None:
            if 'Comics' in error:
                return api.abort(400, error)
            return api.abort(404, error)
        return comic
    
    # def put(self, base_id, merging_id):
    #     '''Merge two comics by their respective id'''
    #     return self.patch(base_id, merging_id)

## Route put option exposed but not available in swagger
@server.route('/comics/<int:comic_id>/<int:comic_merging_id>/', methods=['PUT'])
def merge_comics_by_id(comic_id, comic_merging_id):
    comic, error = merge_comics(comic_id, comic_merging_id)
    if error != None:
        if 'Comics' in error:
            return error, 400
        return error, 404
    return comic.toJSON()

# API Error handling

@server.errorhandler(404)
def handle_bad_request(e):
    return 'Check the URL used', 404
@server.errorhandler(ValueError)        
def value_error(e):
    server.logger.error(e)
    return 'Bad request, check the data format', 400
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
