# helpers/server.py

def titles_parser(body: dict) -> str:
    err_msg: str = ''
    if 'titles' in body and (
        (type(body['titles']) is not list and type(body['titles']) is not str) or
        body['titles'] == '' or
        (0 == len(body['titles']) or body['titles'][0] == '')
    ):
        err_msg += '[titles]: if set should be a non-empty list or string. '
    return err_msg


def genres_and_publishers_parser(body: dict) -> str:
    err_msg: str = ''
    if 'genres' in body and (type(body['genres']) is not list or
                             '' in body['genres'] or 0 == len(body['genres'])):
        err_msg += '[genres]: if set should be a non-empty list. '
    if 'published_in' in body and (type(body['published_in']) is not list or
                                   '' in body['published_in'] or 0 == len(body['published_in'])):
        err_msg += '[published_in]: if set should be a non-empty list. '
    return err_msg


def cover_parser(body: dict) -> str:
    err_msg: str = ''
    if 'cover' not in body or 0 == len(body['cover']):
        return err_msg
    if type(body['cover']) is not str:
        err_msg += '[cover]: if set should be a string. '
    if "http" not in body['cover']:
        err_msg += '[cover]: if set should be a correct http link. '
    return err_msg


def general_field_parser(body, field, _type):
    if field in body and _type == int:
        try:
            int(body[field])
            return ''
        except ValueError:
            return f'[{field}]: if set should be a {_type}. '
    if field in body and type(body[field]) is not _type:
        return f'[{field}]: if set should be a {_type}. '
    return ''


def put_body_parser(json_body: dict) -> str:
    if type(json_body) is not dict:
        return 'Impossible to read json body. '
    err_msg: str = ''
    err_msg += titles_parser(json_body)
    err_msg += genres_and_publishers_parser(json_body)
    err_msg += cover_parser(json_body)
    err_msg += general_field_parser(json_body, 'author', str)
    err_msg += general_field_parser(json_body, 'description', str)
    err_msg += general_field_parser(json_body, 'track', bool)
    err_msg += general_field_parser(json_body, 'viewed_chap', int)
    err_msg += general_field_parser(json_body, 'com_type', int)
    err_msg += general_field_parser(json_body, 'status', int)
    err_msg += general_field_parser(json_body, 'rating', int)
    return err_msg
