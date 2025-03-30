package main

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"net"
	"sync"
	"time"

	"crypto/rand"
	"math/big"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/timestamppb"

	desc "github.com/HpPpL/microservices_course_auth/grpc/pkg/auth_v1"
)

const grpcPort = 50051

type User struct {
	Id        int64
	Name      string
	Email     string
	Role      desc.Role
	CreatedAt time.Time
	UpdatedAt time.Time
}

type server struct {
	desc.UnimplementedAuthV1Server
}

type SyncMap struct {
	elems map[int64]*User
	m     sync.RWMutex
}

var users = &SyncMap{
	elems: make(map[int64]*User),
}

var (
	// Create errors
	passwordDontMatch  = errors.New("password don't match")
	idGenerationFailed = errors.New("id generation failed")

	// Get errors
	userDontExist = errors.New("user dont exist")
)

// Create part
func (s *server) Create(ctx context.Context, req *desc.CreateRequest) (*desc.CreateResponse, error) {
	log.Print("There is create request!")
	if req.GetInfo().GetPassword() != req.GetInfo().GetPasswordConfirm() {
		log.Print("Passwords do not match!")
		return &desc.CreateResponse{}, passwordDontMatch
	}

	maxNum := big.NewInt(0).Lsh(big.NewInt(1), 63)
	n, err := rand.Int(rand.Reader, maxNum)
	if err != nil {
		log.Print("Id generation failed")
		return &desc.CreateResponse{}, idGenerationFailed
	}
	id := n.Int64()

	now := time.Now()
	user := &User{
		Id:        id,
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
func (s *server) Get(ctx context.Context, req *desc.GetRequest) (*desc.GetResponse, error) {
	id := req.GetId()

	users.m.RLock()
	user, ok := users.elems[id]
	users.m.RUnlock()

	if !ok {
		return &desc.GetResponse{}, userDontExist
	}

	return &desc.GetResponse{
		Id:        user.Id,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}, nil

}

// Update part
func (s *server) Update(ctx context.Context, req *desc.UpdateRequest) (*emptypb.Empty, error) {
	id := req.GetId()

	users.m.Lock()
	defer users.m.Unlock()
	user, ok := users.elems[id]

	if !ok {
		log.Print("User not found!")
		return &emptypb.Empty{}, userDontExist
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
func (s *server) Delete(ctx context.Context, req *desc.DeleteRequest) (*emptypb.Empty, error) {
	id := req.GetId()

	users.m.Lock()
	defer users.m.Unlock()
	_, ok := users.elems[id]

	if !ok {
		return &emptypb.Empty{}, userDontExist
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
