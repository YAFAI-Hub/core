package wsp

import (
	"context"
	db "yafai/internal/bridge/db"
)

type WorkspaceServer struct {
	UnimplementedWorkspaceServiceServer
	Db *db.DBWrapper
	Ctx context.Context

}
