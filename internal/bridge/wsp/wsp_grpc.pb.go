// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.3
// source: internal/bridge/wsp/wsp.proto

package wsp

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	WorkspaceService_LinkStream_FullMethodName            = "/wsp.WorkspaceService/LinkStream"
	WorkspaceService_InvokePlanner_FullMethodName         = "/wsp.WorkspaceService/InvokePlanner"
	WorkspaceService_InvokePlanRefine_FullMethodName      = "/wsp.WorkspaceService/InvokePlanRefine"
	WorkspaceService_InvokeOrchestrator_FullMethodName    = "/wsp.WorkspaceService/InvokeOrchestrator"
	WorkspaceService_InvokeAgent_FullMethodName           = "/wsp.WorkspaceService/InvokeAgent"
	WorkspaceService_MonitorAgentExecution_FullMethodName = "/wsp.WorkspaceService/MonitorAgentExecution"
	WorkspaceService_ExecuteAgent_FullMethodName          = "/wsp.WorkspaceService/ExecuteAgent"
	WorkspaceService_ToolDiscovery_FullMethodName         = "/wsp.WorkspaceService/ToolDiscovery"
	WorkspaceService_ToolExecute_FullMethodName           = "/wsp.WorkspaceService/ToolExecute"
)

// WorkspaceServiceClient is the client API for WorkspaceService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type WorkspaceServiceClient interface {
	LinkStream(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[LinkRequest, LinkResponse], error)
	InvokePlanner(ctx context.Context, in *PlannerRequest, opts ...grpc.CallOption) (*PlannerResponse, error)
	InvokePlanRefine(ctx context.Context, in *PlannerRefineRequest, opts ...grpc.CallOption) (*PlannerResponse, error)
	InvokeOrchestrator(ctx context.Context, in *OrchestratorRequest, opts ...grpc.CallOption) (*OrchestratorResponse, error)
	InvokeAgent(ctx context.Context, in *AgentRequest, opts ...grpc.CallOption) (*AgentResponse, error)
	MonitorAgentExecution(ctx context.Context, in *MonitorAgentRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[MonitorAgentResponse], error)
	ExecuteAgent(ctx context.Context, in *AgentRequest, opts ...grpc.CallOption) (*AgentResponse, error)
	ToolDiscovery(ctx context.Context, in *DiscoveryRequest, opts ...grpc.CallOption) (*DiscoveryResponse, error)
	ToolExecute(ctx context.Context, in *ToolExecuteRequest, opts ...grpc.CallOption) (*ToolExecuteResponse, error)
}

type workspaceServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewWorkspaceServiceClient(cc grpc.ClientConnInterface) WorkspaceServiceClient {
	return &workspaceServiceClient{cc}
}

func (c *workspaceServiceClient) LinkStream(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[LinkRequest, LinkResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &WorkspaceService_ServiceDesc.Streams[0], WorkspaceService_LinkStream_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[LinkRequest, LinkResponse]{ClientStream: stream}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type WorkspaceService_LinkStreamClient = grpc.BidiStreamingClient[LinkRequest, LinkResponse]

func (c *workspaceServiceClient) InvokePlanner(ctx context.Context, in *PlannerRequest, opts ...grpc.CallOption) (*PlannerResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(PlannerResponse)
	err := c.cc.Invoke(ctx, WorkspaceService_InvokePlanner_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workspaceServiceClient) InvokePlanRefine(ctx context.Context, in *PlannerRefineRequest, opts ...grpc.CallOption) (*PlannerResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(PlannerResponse)
	err := c.cc.Invoke(ctx, WorkspaceService_InvokePlanRefine_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workspaceServiceClient) InvokeOrchestrator(ctx context.Context, in *OrchestratorRequest, opts ...grpc.CallOption) (*OrchestratorResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(OrchestratorResponse)
	err := c.cc.Invoke(ctx, WorkspaceService_InvokeOrchestrator_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workspaceServiceClient) InvokeAgent(ctx context.Context, in *AgentRequest, opts ...grpc.CallOption) (*AgentResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(AgentResponse)
	err := c.cc.Invoke(ctx, WorkspaceService_InvokeAgent_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workspaceServiceClient) MonitorAgentExecution(ctx context.Context, in *MonitorAgentRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[MonitorAgentResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &WorkspaceService_ServiceDesc.Streams[1], WorkspaceService_MonitorAgentExecution_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[MonitorAgentRequest, MonitorAgentResponse]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type WorkspaceService_MonitorAgentExecutionClient = grpc.ServerStreamingClient[MonitorAgentResponse]

func (c *workspaceServiceClient) ExecuteAgent(ctx context.Context, in *AgentRequest, opts ...grpc.CallOption) (*AgentResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(AgentResponse)
	err := c.cc.Invoke(ctx, WorkspaceService_ExecuteAgent_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workspaceServiceClient) ToolDiscovery(ctx context.Context, in *DiscoveryRequest, opts ...grpc.CallOption) (*DiscoveryResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(DiscoveryResponse)
	err := c.cc.Invoke(ctx, WorkspaceService_ToolDiscovery_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workspaceServiceClient) ToolExecute(ctx context.Context, in *ToolExecuteRequest, opts ...grpc.CallOption) (*ToolExecuteResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ToolExecuteResponse)
	err := c.cc.Invoke(ctx, WorkspaceService_ToolExecute_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// WorkspaceServiceServer is the server API for WorkspaceService service.
// All implementations must embed UnimplementedWorkspaceServiceServer
// for forward compatibility.
type WorkspaceServiceServer interface {
	LinkStream(grpc.BidiStreamingServer[LinkRequest, LinkResponse]) error
	InvokePlanner(context.Context, *PlannerRequest) (*PlannerResponse, error)
	InvokePlanRefine(context.Context, *PlannerRefineRequest) (*PlannerResponse, error)
	InvokeOrchestrator(context.Context, *OrchestratorRequest) (*OrchestratorResponse, error)
	InvokeAgent(context.Context, *AgentRequest) (*AgentResponse, error)
	MonitorAgentExecution(*MonitorAgentRequest, grpc.ServerStreamingServer[MonitorAgentResponse]) error
	ExecuteAgent(context.Context, *AgentRequest) (*AgentResponse, error)
	ToolDiscovery(context.Context, *DiscoveryRequest) (*DiscoveryResponse, error)
	ToolExecute(context.Context, *ToolExecuteRequest) (*ToolExecuteResponse, error)
	mustEmbedUnimplementedWorkspaceServiceServer()
}

// UnimplementedWorkspaceServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedWorkspaceServiceServer struct{}

func (UnimplementedWorkspaceServiceServer) LinkStream(grpc.BidiStreamingServer[LinkRequest, LinkResponse]) error {
	return status.Errorf(codes.Unimplemented, "method LinkStream not implemented")
}
func (UnimplementedWorkspaceServiceServer) InvokePlanner(context.Context, *PlannerRequest) (*PlannerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method InvokePlanner not implemented")
}
func (UnimplementedWorkspaceServiceServer) InvokePlanRefine(context.Context, *PlannerRefineRequest) (*PlannerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method InvokePlanRefine not implemented")
}
func (UnimplementedWorkspaceServiceServer) InvokeOrchestrator(context.Context, *OrchestratorRequest) (*OrchestratorResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method InvokeOrchestrator not implemented")
}
func (UnimplementedWorkspaceServiceServer) InvokeAgent(context.Context, *AgentRequest) (*AgentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method InvokeAgent not implemented")
}
func (UnimplementedWorkspaceServiceServer) MonitorAgentExecution(*MonitorAgentRequest, grpc.ServerStreamingServer[MonitorAgentResponse]) error {
	return status.Errorf(codes.Unimplemented, "method MonitorAgentExecution not implemented")
}
func (UnimplementedWorkspaceServiceServer) ExecuteAgent(context.Context, *AgentRequest) (*AgentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ExecuteAgent not implemented")
}
func (UnimplementedWorkspaceServiceServer) ToolDiscovery(context.Context, *DiscoveryRequest) (*DiscoveryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ToolDiscovery not implemented")
}
func (UnimplementedWorkspaceServiceServer) ToolExecute(context.Context, *ToolExecuteRequest) (*ToolExecuteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ToolExecute not implemented")
}
func (UnimplementedWorkspaceServiceServer) mustEmbedUnimplementedWorkspaceServiceServer() {}
func (UnimplementedWorkspaceServiceServer) testEmbeddedByValue()                          {}

// UnsafeWorkspaceServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to WorkspaceServiceServer will
// result in compilation errors.
type UnsafeWorkspaceServiceServer interface {
	mustEmbedUnimplementedWorkspaceServiceServer()
}

func RegisterWorkspaceServiceServer(s grpc.ServiceRegistrar, srv WorkspaceServiceServer) {
	// If the following call pancis, it indicates UnimplementedWorkspaceServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&WorkspaceService_ServiceDesc, srv)
}

func _WorkspaceService_LinkStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(WorkspaceServiceServer).LinkStream(&grpc.GenericServerStream[LinkRequest, LinkResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type WorkspaceService_LinkStreamServer = grpc.BidiStreamingServer[LinkRequest, LinkResponse]

func _WorkspaceService_InvokePlanner_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PlannerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkspaceServiceServer).InvokePlanner(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WorkspaceService_InvokePlanner_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WorkspaceServiceServer).InvokePlanner(ctx, req.(*PlannerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkspaceService_InvokePlanRefine_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PlannerRefineRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkspaceServiceServer).InvokePlanRefine(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WorkspaceService_InvokePlanRefine_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WorkspaceServiceServer).InvokePlanRefine(ctx, req.(*PlannerRefineRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkspaceService_InvokeOrchestrator_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OrchestratorRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkspaceServiceServer).InvokeOrchestrator(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WorkspaceService_InvokeOrchestrator_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WorkspaceServiceServer).InvokeOrchestrator(ctx, req.(*OrchestratorRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkspaceService_InvokeAgent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AgentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkspaceServiceServer).InvokeAgent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WorkspaceService_InvokeAgent_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WorkspaceServiceServer).InvokeAgent(ctx, req.(*AgentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkspaceService_MonitorAgentExecution_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(MonitorAgentRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(WorkspaceServiceServer).MonitorAgentExecution(m, &grpc.GenericServerStream[MonitorAgentRequest, MonitorAgentResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type WorkspaceService_MonitorAgentExecutionServer = grpc.ServerStreamingServer[MonitorAgentResponse]

func _WorkspaceService_ExecuteAgent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AgentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkspaceServiceServer).ExecuteAgent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WorkspaceService_ExecuteAgent_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WorkspaceServiceServer).ExecuteAgent(ctx, req.(*AgentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkspaceService_ToolDiscovery_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DiscoveryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkspaceServiceServer).ToolDiscovery(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WorkspaceService_ToolDiscovery_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WorkspaceServiceServer).ToolDiscovery(ctx, req.(*DiscoveryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkspaceService_ToolExecute_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ToolExecuteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkspaceServiceServer).ToolExecute(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WorkspaceService_ToolExecute_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WorkspaceServiceServer).ToolExecute(ctx, req.(*ToolExecuteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// WorkspaceService_ServiceDesc is the grpc.ServiceDesc for WorkspaceService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var WorkspaceService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "wsp.WorkspaceService",
	HandlerType: (*WorkspaceServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "InvokePlanner",
			Handler:    _WorkspaceService_InvokePlanner_Handler,
		},
		{
			MethodName: "InvokePlanRefine",
			Handler:    _WorkspaceService_InvokePlanRefine_Handler,
		},
		{
			MethodName: "InvokeOrchestrator",
			Handler:    _WorkspaceService_InvokeOrchestrator_Handler,
		},
		{
			MethodName: "InvokeAgent",
			Handler:    _WorkspaceService_InvokeAgent_Handler,
		},
		{
			MethodName: "ExecuteAgent",
			Handler:    _WorkspaceService_ExecuteAgent_Handler,
		},
		{
			MethodName: "ToolDiscovery",
			Handler:    _WorkspaceService_ToolDiscovery_Handler,
		},
		{
			MethodName: "ToolExecute",
			Handler:    _WorkspaceService_ToolExecute_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "LinkStream",
			Handler:       _WorkspaceService_LinkStream_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "MonitorAgentExecution",
			Handler:       _WorkspaceService_MonitorAgentExecution_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "internal/bridge/wsp/wsp.proto",
}
