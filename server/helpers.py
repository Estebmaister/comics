# server/helpers.py

def titles_parser(body: dict) -> str:
    err_msg: str = ''
    if 'titles' in body and (type(body['titles']) != list or 
        '' in body['titles'] or 0 == len(body['titles'])):
        err_msg +=  'titles: if set should be a non-empty list. '
    return err_msg

def genres_and_publishers_parser(body: dict) -> str:
    err_msg: str = ''
    if 'genres' in body and (type(body['genres']) != list or
        '' in body['genres'] or 0 == len(body['genres'])):
        err_msg += 'genres: if set should be a non-empty list. '
    if 'published_in' in body and (type(body['published_in']) != list or
        '' in body['published_in'] or 0 == len(body['published_in'])):
        err_msg += 'published_in: if set should be a non-empty list. '
    return err_msg

def general_field_parser(body, field, _type):
    if field in body and _type == int:
        try:
            int(body[field])
            return ''
        except ValueError:
            return f'[{field}]: if set should be a {_type}. '
    if field in body and type(body[field]) != _type:
        return f'[{field}]: if set should be a {_type}. '
    return ''

def put_body_parser(json_body: dict) -> str:
    if type(json_body) != dict: return 'Impossible to read json body. '
    err_msg: str = ''
    err_msg += titles_parser(json_body)
    err_msg += genres_and_publishers_parser(json_body)
    err_msg += general_field_parser(json_body, 'author', str)
    err_msg += general_field_parser(json_body, 'description', str)
    err_msg += general_field_parser(json_body, 'track', bool)
    err_msg += general_field_parser(json_body, 'viewed_chap', int)
    err_msg += general_field_parser(json_body, 'com_type', int)
    err_msg += general_field_parser(json_body, 'status', int)
    if ('cover' in json_body and (type(json_body['cover']) is not str or 
        "http" not in json_body['cover'])):
        err_msg += '[cover]: if set should be a correct http link. '
    return err_msg