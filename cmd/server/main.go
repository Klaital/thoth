package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	migmysql "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/klaital/thoth/internal/auth"
	"github.com/klaital/thoth/internal/db"
	"github.com/klaital/thoth/db/migrations"
	thothv1 "github.com/klaital/thoth/internal/gen/thoth/v1"
	"github.com/klaital/thoth/internal/handler"
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

	dsn         := mustEnv("DB_DSN")
	jwtSecret   := mustEnv("JWT_SECRET")
	googleID    := mustEnv("GOOGLE_CLIENT_ID")
	googleSec   := mustEnv("GOOGLE_CLIENT_SECRET")
	googleRedir := mustEnv("GOOGLE_REDIRECT_URL")

	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("opening DB: %v", err)
	}
	defer sqlDB.Close()
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("pinging DB: %v", err)
	}
	if err := runMigrations(dsn); err != nil {
		log.Fatalf("running migrations: %v", err)
	}

	queries := db.New(sqlDB)
	jwtSvc := auth.NewJWTService(jwtSecret)
	googleOAuth := auth.NewGoogleOAuth(googleID, googleSec, googleRedir)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(authInterceptor(jwtSvc)),
	)
	thothv1.RegisterUserServiceServer(grpcServer, handler.NewUserHandler(queries, jwtSvc))
	thothv1.RegisterCollectionServiceServer(grpcServer, handler.NewCollectionHandler(queries))
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

	ctx := context.Background()
	gwMux := runtime.NewServeMux()
	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	if err := thothv1.RegisterUserServiceHandlerFromEndpoint(ctx, gwMux, *grpcAddr, dialOpts); err != nil {
		log.Fatalf("registering user gateway: %v", err)
	}
	if err := thothv1.RegisterCollectionServiceHandlerFromEndpoint(ctx, gwMux, *grpcAddr, dialOpts); err != nil {
		log.Fatalf("registering collection gateway: %v", err)
	}

	oauthHandlers := handler.NewOAuthHandlers(queries, jwtSvc, googleOAuth, *frontendURL)

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/auth/google", oauthHandlers.Login)
	httpMux.HandleFunc("/auth/google/callback", oauthHandlers.Callback)
	httpMux.Handle("/", gwMux)

	log.Printf("HTTP listening on %s", *httpAddr)
	if err := http.ListenAndServe(*httpAddr, httpMux); err != nil {
		log.Fatalf("HTTP serve: %v", err)
	}
}

func runMigrations(dsn string) error {
	// Open a separate connection with multiStatements=true — required to execute
	// migration files that contain more than one statement.
	if strings.Contains(dsn, "?") {
		dsn += "&multiStatements=true"
	} else {
		dsn += "?multiStatements=true"
	}
	migDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer migDB.Close()

	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}
	driver, err := migmysql.WithInstance(migDB, &migmysql.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", src, "mysql", driver)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func authInterceptor(jwtSvc *auth.JWTService) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}
		token, err := auth.ExtractBearerToken(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "missing or invalid authorization header")
		}
		claims, err := jwtSvc.Validate(token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}
		ctx = auth.ContextWithUserID(ctx, claims.UserID)
		return handler(ctx, req)
	}
}
