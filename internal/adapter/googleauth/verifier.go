package googleauth

import (
	"context"
	"fmt"
	"strings"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"google.golang.org/api/idtoken"
)

type Verifier struct {
	clientID string
}

func NewVerifier(clientID string) *Verifier {
	return &Verifier{clientID: clientID}
}

func (v *Verifier) Verify(ctx context.Context, rawToken string) (*domain.GoogleIdentity, error) {
	payload, err := idtoken.Validate(ctx, rawToken, v.clientID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidGoogleToken, err)
	}
	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	if email == "" || payload.Subject == "" {
		return nil, domain.ErrInvalidGoogleToken
	}
	return &domain.GoogleIdentity{
		Sub:   payload.Subject,
		Email: strings.TrimSpace(email),
		Name:  strings.TrimSpace(name),
	}, nil
}
