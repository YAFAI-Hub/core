package wsp

import (
	"context"
	"yafai/internal/nexus/workspace"
)

type WorkspaceServer struct {
	UnimplementedWorkspaceServiceServer
	Wsp *workspace.Workspace
	Ctx context.Context
}
