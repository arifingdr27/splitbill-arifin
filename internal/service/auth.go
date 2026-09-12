package service

import (
	"context"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/arifin2018/splitbill-arifin.git/internal/port"
	"github.com/sirupsen/logrus"
)

type AuthService struct {
	users    port.UserRepository
	verifier port.GoogleTokenVerifier
	tokens   port.TokenIssuer
	log      *logrus.Logger
}

func NewAuthService(users port.UserRepository, verifier port.GoogleTokenVerifier, tokens port.TokenIssuer, log *logrus.Logger) *AuthService {
	return &AuthService{users: users, verifier: verifier, tokens: tokens, log: log}
}

func (s *AuthService) LoginWithGoogle(ctx context.Context, idToken string) (*domain.AuthResult, error) {
	identity, err := s.verifier.Verify(ctx, idToken)
	if err != nil {
		s.log.WithError(err).Warn("google token verify failed")
		return nil, domain.ErrInvalidGoogleToken
	}
	user, err := s.users.UpsertGoogleUser(ctx, *identity)
	if err != nil {
		s.log.WithError(err).Error("upsert user failed")
		return nil, err
	}
	token, err := s.tokens.Issue(user)
	if err != nil {
		return nil, err
	}
	return &domain.AuthResult{
		Token: token,
		User: domain.AuthUser{
			ID:    user.ID.String(),
			Email: user.Email,
			Name:  user.Name,
		},
	}, nil
}
