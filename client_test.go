package mlcartifact

import (
	"context"
	"net"
	"testing"

	pb "github.com/hmsoft0815/mlcartifact/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type mockArtifactServer struct {
	pb.UnimplementedArtifactServiceServer
	lastWrite *pb.WriteRequest
}

func (m *mockArtifactServer) Write(ctx context.Context, req *pb.WriteRequest) (*pb.WriteResponse, error) {
	m.lastWrite = req
	return &pb.WriteResponse{
		Id:       "test-id",
		Filename: req.Filename,
		Uri:      "artifact://test-id",
	}, nil
}

func (m *mockArtifactServer) Read(ctx context.Context, req *pb.ReadRequest) (*pb.ReadResponse, error) {
	return &pb.ReadResponse{
		Content:  []byte("read-data"),
		Filename: "read.txt",
		MimeType: "text/plain",
	}, nil
}

func TestClient(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	mockSrv := &mockArtifactServer{}
	pb.RegisterArtifactServiceServer(s, mockSrv)
	go func() {
		if err := s.Serve(lis); err != nil {
			return
		}
	}()
	defer s.Stop()

	// Create client using the buffered connection
	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	client := &Client{
		conn: conn,
		cli:  pb.NewArtifactServiceClient(conn),
	}

	ctx := context.Background()

	// Test Write
	res, err := client.Write(ctx, "hello.txt", []byte("world"), WithSource("test"))
	require.NoError(t, err)
	assert.Equal(t, "test-id", res.Id)
	assert.Equal(t, "test", mockSrv.lastWrite.Source)

	// Test Read
	readRes, err := client.Read(ctx, "test-id")
	require.NoError(t, err)
	assert.Equal(t, []byte("read-data"), readRes.Content)
}
