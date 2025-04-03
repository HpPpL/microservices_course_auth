package main

import (
	"context"
	"database/sql"

	"google.golang.org/protobuf/types/known/timestamppb"

	"errors"
	"flag"
	"log"
	"net"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/HpPpL/microservices_course_auth/internal/config"
	desc "github.com/HpPpL/microservices_course_auth/pkg/auth_v1"
)

// Path to config
var configPath string

func init() {
	flag.StringVar(&configPath, "config-path", ".env", "path to config file")
}

// User represents a system user with ID, name, email, role, and timestamps for creation and updates.
type User struct {
	ID        int64
	Name      string
	Email     string
	Role      desc.Role
	CreatedAt time.Time
	UpdatedAt sql.NullTime
}

// server implements the AuthV1 gRPC service by embedding UnimplementedAuthV1Server.
type server struct {
	desc.UnimplementedAuthV1Server
	pool *pgxpool.Pool
}

var (
	// General PG errors
	errFailedBuildQuery = errors.New("failed to build query")
	errUserDoesntExist  = errors.New("user with current id doesn't exist")

	// Create errors
	errPasswordDoesntMatch = errors.New("password doesn't match")
	errInvalidRole         = errors.New("invalid role value")
	errFailedInsertUser    = errors.New("failed to insert user")

	// Get errors
	errFailedSelectUser = errors.New("failed to select user")

	// Update errors
	errFailedUpdateUser = errors.New("failed to update user data")

	// Delete errors
	errFailedDeleteUser = errors.New("failed to delete user")
)

const (
	roleUnspecified = "unspecified"
	roleUser        = "user"
	roleAdmin       = "admin"
)

// Create part
func (s *server) Create(ctx context.Context, req *desc.CreateRequest) (*desc.CreateResponse, error) {

	log.Print("There is create request!")
	if req.GetInfo().GetPassword() != req.GetInfo().GetPasswordConfirm() {
		log.Print("Passwords do not match!")
		return &desc.CreateResponse{}, errPasswordDoesntMatch
	}

	var roleStr string
	switch req.GetInfo().GetRole() {
	case desc.Role_ROLE_UNSPECIFIED:
		roleStr = roleUnspecified
	case desc.Role_ROLE_USER:
		roleStr = roleUser
	case desc.Role_ROLE_ADMIN:
		roleStr = roleAdmin
	default:
		log.Printf("invalid role value: %v", req.GetInfo().GetRole())
		return &desc.CreateResponse{}, errInvalidRole
	}

	// Вынести имя таблицы можно попробовать потом
	builderInsert := sq.Insert("users").
		PlaceholderFormat(sq.Dollar).
		Columns("name", "email", "role").
		Values(req.GetInfo().GetName(), req.GetInfo().GetEmail(), roleStr).
		Suffix("RETURNING id")

	query, args, err := builderInsert.ToSql()
	if err != nil {
		log.Printf("failed to build query: %v", err)
		return &desc.CreateResponse{}, errFailedBuildQuery
	}

	var UserID int64
	err = s.pool.QueryRow(ctx, query, args...).Scan(&UserID)
	if err != nil {
		log.Printf("failed to insert user: %v", err)
		return &desc.CreateResponse{}, errFailedInsertUser
	}

	log.Printf("inserted user with id: %v", UserID)
	return &desc.CreateResponse{
		Id: UserID,
	}, nil
}

// Get part
func (s *server) Get(ctx context.Context, req *desc.GetRequest) (*desc.GetResponse, error) {
	log.Print("There is get request!")

	builderSelectOne := sq.Select("id", "name", "email", "role", "created_at", "updated_at").
		From("users").
		PlaceholderFormat(sq.Dollar).
		Where(sq.Eq{"id": req.GetId()}).
		Limit(1)

	query, args, err := builderSelectOne.ToSql()
	if err != nil {
		log.Printf("failed to build query: %v", err)
		return &desc.GetResponse{}, errFailedBuildQuery
	}

	var id int64
	var name, email, roleStr string
	var createdAt time.Time
	var updatedAt sql.NullTime

	err = s.pool.QueryRow(ctx, query, args...).Scan(&id, &name, &email, &roleStr, &createdAt, &updatedAt)
	if err != nil {
		log.Printf("failed to select user: %v", err)
		return &desc.GetResponse{}, errFailedSelectUser
	}

	var role desc.Role
	switch roleStr {
	case roleUnspecified:
		role = desc.Role_ROLE_UNSPECIFIED
	case roleUser:
		role = desc.Role_ROLE_USER
	case roleAdmin:
		role = desc.Role_ROLE_ADMIN
	default:
		log.Print(errInvalidRole.Error())
		return &desc.GetResponse{}, errInvalidRole
	}

	log.Printf("ID: %v, Name: %v, Email: %v, Role: %v, CreatedAt: %v, UpdatedAt: %v",
		id, name, email, role, createdAt, updatedAt)

	var updatedAtTime *timestamppb.Timestamp
	if updatedAt.Valid {
		updatedAtTime = timestamppb.New(updatedAt.Time)
	}

	return &desc.GetResponse{
		Id:        id,
		Name:      name,
		Email:     email,
		Role:      role,
		CreatedAt: timestamppb.New(createdAt),
		UpdatedAt: updatedAtTime,
	}, nil
}

// Update part
func (s *server) Update(ctx context.Context, req *desc.UpdateRequest) (*emptypb.Empty, error) {
	log.Print("There is update request!")

	userID := req.GetId()

	builderUpdate := sq.Update("users").
		PlaceholderFormat(sq.Dollar).
		Where(sq.Eq{"id": userID})

	nameWrapper := req.GetName()
	if nameWrapper != nil {
		builderUpdate = builderUpdate.Set("name", nameWrapper.GetValue())
	}

	emailWrapper := req.GetEmail()
	if emailWrapper != nil {
		builderUpdate = builderUpdate.Set("email", emailWrapper.GetValue())
	}

	builderUpdate = builderUpdate.Set("updated_at", time.Now())

	query, args, err := builderUpdate.ToSql()
	if err != nil {
		log.Printf("failed to build query: %v", err)
		return &emptypb.Empty{}, errFailedBuildQuery
	}

	res, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		log.Printf("failed to update user data: %v", err)
		return &emptypb.Empty{}, errFailedUpdateUser
	}

	if res.RowsAffected() == 0 {
		return &emptypb.Empty{}, errUserDoesntExist
	}

	log.Printf("updated %d rows", res.RowsAffected())

	return &emptypb.Empty{}, nil
}

// Delete part
func (s *server) Delete(ctx context.Context, req *desc.DeleteRequest) (*emptypb.Empty, error) {
	userID := req.GetId()

	builderDelete := sq.Delete("users").
		PlaceholderFormat(sq.Dollar).
		Where(sq.Eq{"id": userID})

	query, args, err := builderDelete.ToSql()
	if err != nil {
		log.Printf("failed to build query: %v", err)
		return &emptypb.Empty{}, errFailedBuildQuery
	}

	res, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		log.Printf("failed to delete user: %v", err)
		return &emptypb.Empty{}, errFailedDeleteUser
	}

	if res.RowsAffected() == 0 {
		return &emptypb.Empty{}, errUserDoesntExist
	}

	log.Printf("Deleted %d rows", res.RowsAffected())
	return &emptypb.Empty{}, nil
}

func main() {
	flag.Parse()
	ctx := context.Background()

	err := config.Load(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	grpcConfig, err := config.NewGRPCConfig()
	if err != nil {
		log.Fatalf("failed to get grpc config")
	}

	pgConfig, err := config.NewPGConfig()
	if err != nil {
		log.Fatalf("failed to get pg config")
	}

	lis, err := net.Listen("tcp", grpcConfig.Address())
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	pool, err := pgxpool.Connect(ctx, pgConfig.DSN())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	s := grpc.NewServer()
	reflection.Register(s)
	desc.RegisterAuthV1Server(s, &server{pool: pool})

	log.Printf("server listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
