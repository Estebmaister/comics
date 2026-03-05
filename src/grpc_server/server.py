import logging
from concurrent import futures

import grpc

from grpc_server.auth_service import AuthService
from grpc_server.comics_service import ComicsService
from pb import auth_pb2_grpc
from pb import comics_service_pb2_grpc


def create_server() -> grpc.Server:
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    auth_pb2_grpc.add_AuthServiceServicer_to_server(AuthService(), server)
    comics_service_pb2_grpc.add_ComicServiceServicer_to_server(ComicsService(), server)
    return server


def serve(port: int = 50051) -> None:
    logging.basicConfig(level=logging.INFO)
    server = create_server()
    server.add_insecure_port(f"[::]:{port}")
    logging.info("gRPC server starting on port %s", port)
    server.start()
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
