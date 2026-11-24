package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/joho/godotenv"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/klaital/thoth/db/migrations"
	"github.com/klaital/thoth/internal/auth"
	"github.com/klaital/thoth/internal/db"
	thothv1 "github.com/klaital/thoth/internal/gen/thoth/v1"
	"github.com/klaital/thoth/internal/goupc"
	"github.com/klaital/thoth/internal/handler"
	"github.com/klaital/thoth/internal/telemetry"
)

// publicMethods bypass JWT validation.
var publicMethods = map[string]bool{
	"/thoth.v1.UserService/Register":     true,
	"/thoth.v1.UserService/Login":        true,
	"/thoth.v1.UserService/RefreshToken": true,
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s is required", key)
	}
	return v
}

func main() {
	// Load .env if present; ignore error so production (env vars already set) works fine.
	_ = godotenv.Load()

	var (
		grpcAddr    = flag.String("grpc-addr", ":9090", "gRPC listen address")
		httpAddr    = flag.String("http-addr", ":8080", "HTTP/gRPC-Gateway listen address")
		frontendURL = flag.String("frontend-url", "http://localhost:5173", "Frontend base URL for OAuth redirects")
	)
	flag.Parse()

	// Stop on SIGINT/SIGTERM (docker stop) so servers drain and telemetry flushes.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownTelemetry, err := telemetry.Setup(ctx, "thoth-api")
	if err != nil {
		log.Fatalf("setting up telemetry: %v", err)
	}

	dsn := mustEnv("DB_DSN")
	jwtSecret := mustEnv("JWT_SECRET")
	googleID := mustEnv("GOOGLE_CLIENT_ID")
	googleSec := mustEnv("GOOGLE_CLIENT_SECRET")
	googleRedir := mustEnv("GOOGLE_REDIRECT_URL")
	goupcKey := os.Getenv("GOUPC_API_KEY") // optional; UPC lookups disabled if empty

	sqlDB, err := db.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("connecting to DB: %v", err)
	}
	defer sqlDB.Close()
	if err := migrations.Up(dsn); err != nil {
		log.Fatalf("running migrations: %v", err)
	}

	queries := db.New(sqlDB)
	if err := telemetry.RegisterUsageGauges(func(ctx context.Context) (telemetry.UsageTotals, error) {
		t, err := queries.GetUsageTotals(ctx)
		return telemetry.UsageTotals(t), err
	}); err != nil {
		log.Fatalf("registering usage gauges: %v", err)
	}
	jwtSvc := auth.NewJWTService(jwtSecret)
	googleOAuth := auth.NewGoogleOAuth(googleID, googleSec, googleRedir)

	var upcClient *goupc.Client
	if goupcKey != "" {
		upcClient = goupc.New(goupcKey)
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(authInterceptor(jwtSvc)),
		grpc.StreamInterceptor(authStreamInterceptor(jwtSvc)),
	)
	thothv1.RegisterUserServiceServer(grpcServer, handler.NewUserHandler(queries, jwtSvc))
	thothv1.RegisterCollectionServiceServer(grpcServer, handler.NewCollectionHandler(sqlDB, queries))
	thothv1.RegisterCardItemServiceServer(grpcServer, handler.NewCardItemHandler(sqlDB, queries))
	thothv1.RegisterDeckServiceServer(grpcServer, handler.NewDeckHandler(sqlDB, queries))
	thothv1.RegisterSealedItemServiceServer(grpcServer, handler.NewSealedItemHandler(queries, upcClient))
	thothv1.RegisterBookDvdItemServiceServer(grpcServer, handler.NewBookDvdItemHandler(queries))
	thothv1.RegisterScryfallServiceServer(grpcServer, handler.NewScryfallHandler(queries))
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", *grpcAddr)
	if err != nil {
		log.Fatalf("listening on %s: %v", *grpcAddr, err)
	}
	go func() {
		log.Printf("gRPC listening on %s", *grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC serve: %v", err)
		}
	}()

	gwMux := runtime.NewServeMux()
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// Propagates the HTTP request's trace into the gRPC call.
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}

	if err := thothv1.RegisterUserServiceHandlerFromEndpoint(ctx, gwMux, *grpcAddr, dialOpts); err != nil {
		log.Fatalf("registering user gateway: %v", err)
	}
	if err := thothv1.RegisterCollectionServiceHandlerFromEndpoint(ctx, gwMux, *grpcAddr, dialOpts); err != nil {
		log.Fatalf("registering collection gateway: %v", err)
	}
	if err := thothv1.RegisterCardItemServiceHandlerFromEndpoint(ctx, gwMux, *grpcAddr, dialOpts); err != nil {
		log.Fatalf("registering card item gateway: %v", err)
	}
	if err := thothv1.RegisterDeckServiceHandlerFromEndpoint(ctx, gwMux, *grpcAddr, dialOpts); err != nil {
		log.Fatalf("registering deck gateway: %v", err)
	}
	if err := thothv1.RegisterSealedItemServiceHandlerFromEndpoint(ctx, gwMux, *grpcAddr, dialOpts); err != nil {
		log.Fatalf("registering sealed item gateway: %v", err)
	}
	if err := thothv1.RegisterBookDvdItemServiceHandlerFromEndpoint(ctx, gwMux, *grpcAddr, dialOpts); err != nil {
		log.Fatalf("registering book/dvd item gateway: %v", err)
	}
	if err := thothv1.RegisterScryfallServiceHandlerFromEndpoint(ctx, gwMux, *grpcAddr, dialOpts); err != nil {
		log.Fatalf("registering scryfall gateway: %v", err)
	}

	oauthHandlers := handler.NewOAuthHandlers(queries, jwtSvc, googleOAuth, *frontendURL)

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/auth/google", oauthHandlers.Login)
	httpMux.HandleFunc("/auth/google/callback", oauthHandlers.Callback)
	httpMux.Handle("/", noProxyBuffering(gwMux))

	httpServer := &http.Server{
		Addr: *httpAddr,
		Handler: otelhttp.NewHandler(httpMux, "http",
			// Gateway paths embed IDs, so name those spans by method only;
			// the child gRPC span carries the RPC name.
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				if strings.HasPrefix(r.URL.Path, "/auth/") {
					return r.Method + " " + r.URL.Path
				}
				return r.Method
			}),
		),
	}
	serveErr := make(chan error, 1)
	go func() {
		log.Printf("HTTP listening on %s", *httpAddr)
		serveErr <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP serve: %v", err)
		}
	case <-ctx.Done():
		log.Printf("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown: %v", err)
	}
	// Long bulk-import streams shouldn't block exit past the deadline.
	stopped := make(chan struct{})
	go func() { grpcServer.GracefulStop(); close(stopped) }()
	select {
	case <-stopped:
	case <-shutdownCtx.Done():
		grpcServer.Stop()
	}

	// Flushed here rather than deferred: log.Fatalf skips defers anyway, and
	// the final metric export must run before the DB pool closes.
	flushCtx, cancelFlush := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFlush()
	if err := shutdownTelemetry(flushCtx); err != nil {
		log.Printf("telemetry shutdown: %v", err)
	}
}

func authInterceptor(jwtSvc *auth.JWTService) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}
		ctx, err := authenticate(ctx, jwtSvc)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func authStreamInterceptor(jwtSvc *auth.JWTService) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if publicMethods[info.FullMethod] {
			return handler(srv, ss)
		}
		ctx, err := authenticate(ss.Context(), jwtSvc)
		if err != nil {
			return err
		}
		return handler(srv, &authedStream{ServerStream: ss, ctx: ctx})
	}
}

// authedStream overrides a stream's context with one carrying the caller's user ID.
type authedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authedStream) Context() context.Context { return s.ctx }

// authenticate validates the request's bearer token and returns ctx with the
// caller's user ID attached.
func authenticate(ctx context.Context, jwtSvc *auth.JWTService) (context.Context, error) {
	token, err := auth.ExtractBearerToken(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid authorization header")
	}
	claims, err := jwtSvc.Validate(token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
	}
	trace.SpanFromContext(ctx).SetAttributes(attribute.Int64("enduser.id", claims.UserID))
	return auth.ContextWithUserID(ctx, claims.UserID), nil
}

// noProxyBuffering tells nginx to pass responses through as they're written,
// so streamed RPCs (e.g. card bulk import progress) arrive incrementally.
func noProxyBuffering(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Accel-Buffering", "no")
		next.ServeHTTP(w, r)
	})
}
