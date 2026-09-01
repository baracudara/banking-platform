// Package grpcserver adapts the AccountService business logic to the gRPC
// transport. The generated stubs it imports live in the shared `protos`
// module, not in this service, because a proto file is a contract between
// two parties - keeping it here would force every consumer to depend on
// this service's whole module.
package grpcserver

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	accountpb "banking/protos/gen/go/account/v1"
	commonpb "banking/protos/gen/go/common/v1"

	"banking/services/account/internal/domain"
	"banking/services/account/internal/service"
)

type Server struct {
	accountpb.UnimplementedAccountServiceServer
	svc *service.AccountService
}

func NewServer(svc *service.AccountService) *Server {
	return &Server{svc: svc}
}

func (s *Server) Register(grpcServer *grpc.Server) {
	accountpb.RegisterAccountServiceServer(grpcServer, s)
}

func (s *Server) CreateAccount(ctx context.Context, req *accountpb.CreateAccountRequest) (*accountpb.CreateAccountResponse, error) {
	acc, err := s.svc.CreateAccount(ctx, req.GetUserId(), domain.Currency(req.GetCurrency()))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &accountpb.CreateAccountResponse{Account: toProto(acc)}, nil
}

func (s *Server) GetAccount(ctx context.Context, req *accountpb.GetAccountRequest) (*accountpb.GetAccountResponse, error) {
	acc, err := s.svc.GetAccount(ctx, req.GetId())
	if err != nil {
		// errors.Is, not ==: the service layer wraps repository errors with
		// %w, so a direct comparison would never match and every missing
		// account would surface as Internal instead of NotFound.
		if errors.Is(err, domain.ErrAccountNotFound) {
			return nil, status.Error(codes.NotFound, "account not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &accountpb.GetAccountResponse{Account: toProto(acc)}, nil
}

func toProto(acc *domain.Account) *accountpb.Account {
	return &accountpb.Account{
		Id:     acc.ID,
		UserId: acc.UserID,
		Balance: &commonpb.Money{
			AmountMinor:  acc.BalanceCache,
			CurrencyCode: string(acc.Currency),
		},
		Version: acc.Version,
	}
}
