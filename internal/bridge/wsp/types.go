package wsp

import (
	"yafai/internal/nexus/workspace"
)

type WorkspaceServer struct {
	UnimplementedWorkspaceServiceServer
	Wsp *workspace.Workspace
}
