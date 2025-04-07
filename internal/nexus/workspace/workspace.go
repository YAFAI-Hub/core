package workspace

import "log/slog"

func (w *Workspace) AttachVectorStore(vector_store string) (workspace *Workspace) {
	w.VectorStore = vector_store
	slog.Info("Attached Vector store!")
	return w
}
