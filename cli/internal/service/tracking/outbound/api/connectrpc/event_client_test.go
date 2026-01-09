package connectrpc_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/team-attention/cops/cli/internal/platform/setup/config"
	"github.com/team-attention/cops/cli/internal/platform/setup/httpclient"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc"
	eventv1 "github.com/team-attention/cops/shared/gen/grpcstub/event/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/event/v1/eventv1connect"
)

func TestEventAPIClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EventAPIClient Suite")
}

// mockEventService implements eventv1connect.EventServiceHandler for testing.
type mockEventService struct {
	eventv1connect.UnimplementedEventServiceHandler
	sendEventsFunc func(ctx context.Context, req *connect.Request[eventv1.SendEventsReq]) (*connect.Response[eventv1.SendEventsRes], error)
}

func (m *mockEventService) SendEvents(ctx context.Context, req *connect.Request[eventv1.SendEventsReq]) (*connect.Response[eventv1.SendEventsRes], error) {
	if m.sendEventsFunc != nil {
		return m.sendEventsFunc(ctx, req)
	}
	return connect.NewResponse(&eventv1.SendEventsRes{}), nil
}

// createTestClient creates a test EventAPIClient with a mock server.
func createTestClient(serverURL string) *connectrpc.EventAPIClient {
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{
		API: config.APIConfig{
			URL: serverURL,
		},
	}
	httpClient := httpclient.InitAPIHTTPClient(cfg)
	return connectrpc.NewEventAPIClient(l, cfg, httpClient)
}

var _ = Describe("EventAPIClient", func() {
	Describe("SendEvents", func() {
		Context("when server returns success", func() {
			It("returns nil error", func() {
				// 1. Create mock server that returns success.
				mock := &mockEventService{}
				path, handler := eventv1connect.NewEventServiceHandler(mock)
				mux := http.NewServeMux()
				mux.Handle(path, handler)
				server := httptest.NewServer(mux)
				defer server.Close()

				// 2. Create client and send events.
				client := createTestClient(server.URL)
				events := []string{`{"event":"test"}`}
				err := client.SendEvents(context.Background(), "test-api-key", events)

				// 3. Assert no error.
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when checking authorization header", func() {
			It("sets Bearer token correctly", func() {
				// 1. Create mock that captures header.
				var capturedAuthHeader string
				mock := &mockEventService{
					sendEventsFunc: func(ctx context.Context, req *connect.Request[eventv1.SendEventsReq]) (*connect.Response[eventv1.SendEventsRes], error) {
						capturedAuthHeader = req.Header().Get("Authorization")
						return connect.NewResponse(&eventv1.SendEventsRes{}), nil
					},
				}
				path, handler := eventv1connect.NewEventServiceHandler(mock)
				mux := http.NewServeMux()
				mux.Handle(path, handler)
				server := httptest.NewServer(mux)
				defer server.Close()

				// 2. Create client and send events.
				client := createTestClient(server.URL)
				apiKey := "my-secret-api-key-123"
				err := client.SendEvents(context.Background(), apiKey, []string{`{"event":"test"}`})

				// 3. Assert correct authorization header.
				Expect(err).NotTo(HaveOccurred())
				Expect(capturedAuthHeader).To(Equal("Bearer " + apiKey))
			})
		})

		Context("when server is unreachable", func() {
			It("returns network error", func() {
				// 1. Create client with invalid URL.
				client := createTestClient("http://localhost:99999")

				// 2. Send events.
				events := []string{`{"event":"test"}`}
				err := client.SendEvents(context.Background(), "test-api-key", events)

				// 3. Assert error returned.
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when server returns error code", func() {
			It("returns server error", func() {
				// 1. Create mock that returns error.
				mock := &mockEventService{
					sendEventsFunc: func(ctx context.Context, req *connect.Request[eventv1.SendEventsReq]) (*connect.Response[eventv1.SendEventsRes], error) {
						return nil, connect.NewError(connect.CodeInternal, nil)
					},
				}
				path, handler := eventv1connect.NewEventServiceHandler(mock)
				mux := http.NewServeMux()
				mux.Handle(path, handler)
				server := httptest.NewServer(mux)
				defer server.Close()

				// 2. Create client and send events.
				client := createTestClient(server.URL)
				events := []string{`{"event":"test"}`}
				err := client.SendEvents(context.Background(), "test-api-key", events)

				// 3. Assert error returned.
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
