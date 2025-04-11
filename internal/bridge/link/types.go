package link

import "yafai/internal/bridge/wsp"

type LinkServer struct {
	UnimplementedChatServiceServer
	WspStream wsp.WorkspaceService_LinkStreamClient
}
