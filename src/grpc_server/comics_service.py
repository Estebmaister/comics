from datetime import datetime, timezone

from google.protobuf.timestamp_pb2 import Timestamp

from pb import comics_service_pb2
from pb import comics_service_pb2_grpc


def _now_timestamp() -> Timestamp:
    ts = Timestamp()
    ts.FromDatetime(datetime.now(timezone.utc))
    return ts


def _metadata(request_id: str = "", status_code: int = 501, message: str = "Not implemented"):
    return comics_service_pb2.ResponseMetadata(
        request_id=request_id,
        start_time=_now_timestamp(),
        end_time=_now_timestamp(),
        status_code=status_code,
        status_message=message,
    )


class ComicsService(comics_service_pb2_grpc.ComicServiceServicer):
    def CreateComic(self, request, context):
        return comics_service_pb2.ComicResponse(
            metadata=_metadata(request.metadata.request_id if request.metadata else "")
        )

    def DeleteComic(self, request, context):
        return comics_service_pb2.ComicResponse(
            metadata=_metadata(request.metadata.request_id if request.metadata else "")
        )

    def UpdateComic(self, request, context):
        return comics_service_pb2.ComicResponse(
            metadata=_metadata(request.metadata.request_id if request.metadata else "")
        )

    def GetComicById(self, request, context):
        return comics_service_pb2.ComicResponse(
            metadata=_metadata(request.metadata.request_id if request.metadata else "")
        )

    def GetComicByTitle(self, request, context):
        return comics_service_pb2.ComicResponse(
            metadata=_metadata(request.metadata.request_id if request.metadata else "")
        )

    def GetComics(self, request, context):
        return comics_service_pb2.ComicsResponse(
            metadata=_metadata(request.metadata.request_id if request.metadata else "")
        )

    def SearchComics(self, request, context):
        return comics_service_pb2.ComicsResponse(
            metadata=_metadata(request.metadata.request_id if request.metadata else "")
        )
