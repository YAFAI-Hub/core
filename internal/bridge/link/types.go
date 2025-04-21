package link

import (
	"google.golang.org/grpc"
)

type LinkServer struct {
	UnimplementedChatServiceServer
	WspConn *grpc.ClientConn
}
