package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"crypto/rand"
	"math/big"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	desc "github.com/HpPpL/microservices_course_auth/pkg/auth_v1"
)

const grpcPort = 50051

// User represents a system user with ID, name, email, role, and timestamps for creation and updates.
type User struct {
	ID        int64
	Name      string
	Email     string
	Role      desc.Role
	CreatedAt time.Time
	UpdatedAt time.Time
}

// server implements the AuthV1 gRPC service by embedding UnimplementedAuthV1Server.
type server struct {
	desc.UnimplementedAuthV1Server
}

// SyncMap is a thread-safe map for storing pointers to User objects keyed by their ID.
type SyncMap struct {
	elems map[int64]*User
	m     sync.RWMutex
}

var users = &SyncMap{
	elems: make(map[int64]*User),
}

var (
	// Create errors
	errPasswordDoesntMatch = errors.New("password doesn't match")
	errIDGenerationFailed  = errors.New("id generation failed")

	// Get errors
	errUserDoesntExist = errors.New("user doesn't exist")
)

// Create part
func (s *server) Create(_ context.Context, req *desc.CreateRequest) (*desc.CreateResponse, error) {
	log.Print("There is create request!")
	if req.GetInfo().GetPassword() != req.GetInfo().GetPasswordConfirm() {
		log.Print("Passwords do not match!")
		return &desc.CreateResponse{}, errPasswordDoesntMatch
	}

	maxNum := big.NewInt(0).Lsh(big.NewInt(1), 63)
	n, err := rand.Int(rand.Reader, maxNum)
	if err != nil {
		log.Print("ID generation failed")
		return &desc.CreateResponse{}, errIDGenerationFailed
	}
	id := n.Int64()

	now := time.Now()
	user := &User{
		ID:        id,
		Name:      req.GetInfo().GetName(),
		Email:     req.GetInfo().GetEmail(),
		Role:      req.GetInfo().GetRole(),
		CreatedAt: now,
		UpdatedAt: now,
	}

	users.m.Lock()
	defer users.m.Unlock()
	users.elems[id] = user

	return &desc.CreateResponse{Id: id}, nil
}

// Get part
func (s *server) Get(_ context.Context, req *desc.GetRequest) (*desc.GetResponse, error) {
	id := req.GetId()

	users.m.RLock()
	user, ok := users.elems[id]
	users.m.RUnlock()

	if !ok {
		return &desc.GetResponse{}, errUserDoesntExist
	}

	return &desc.GetResponse{
		Id:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}, nil

}

// Update part
func (s *server) Update(_ context.Context, req *desc.UpdateRequest) (*emptypb.Empty, error) {
	id := req.GetId()

	users.m.Lock()
	defer users.m.Unlock()
	user, ok := users.elems[id]

	if !ok {
		log.Print("User not found!")
		return &emptypb.Empty{}, errUserDoesntExist
	}

	nameWrapper := req.GetName()
	if nameWrapper != nil {
		// Если поле не пустое - обновляем
		user.Name = nameWrapper.GetValue()
	}

	emailWrapper := req.GetEmail()
	if emailWrapper != nil {
		// Ну тут аналогично
		user.Email = emailWrapper.GetValue()
	}

	now := time.Now()
	user.UpdatedAt = now

	return &emptypb.Empty{}, nil
}

// Delete part
func (s *server) Delete(_ context.Context, req *desc.DeleteRequest) (*emptypb.Empty, error) {
	id := req.GetId()

	users.m.Lock()
	defer users.m.Unlock()
	_, ok := users.elems[id]

	if !ok {
		return &emptypb.Empty{}, errUserDoesntExist
	}

	delete(users.elems, id)
	return &emptypb.Empty{}, nil
}

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	reflection.Register(s)
	desc.RegisterAuthV1Server(s, &server{})

	log.Printf("server listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
