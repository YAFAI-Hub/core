package bridge

import (
	pb "yafai/internal/bridge/proto"
	"yafai/internal/nexus/workspace"
)

type LinkServer struct {
	pb.UnimplementedAgentServer
	pb.UnimplementedOrchestratorServer
	pb.UnimplementedChatServiceServer
	pb.UnimplementedPlannerServer
	Wsp *workspace.Workspace
}
