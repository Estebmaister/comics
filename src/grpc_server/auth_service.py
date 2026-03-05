from pb import auth_pb2
from pb import auth_pb2_grpc


class AuthService(auth_pb2_grpc.AuthServiceServicer):
    def Register(self, request, context):
        return auth_pb2.AuthResponse()

    def Login(self, request, context):
        return auth_pb2.AuthResponse()

    def ValidateToken(self, request, context):
        return auth_pb2.ValidateTokenResponse(valid=False)
