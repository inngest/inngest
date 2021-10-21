package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Account struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Test bool      `json:"test"`

	Identifier AccountIdentifier `json:"identifier"`
}

type AccountIdentifier struct {
	DSNPrefix  string     `json:"dsnPrefix"`
	Domain     *string    `json:"domain"`
	VerifiedAt *time.Time `json:"verifiedAt"`
}

func (c httpClient) Account(ctx context.Context) (*Account, error) {
	query := `
          query {
            account {
	      id name billingEmail createdAt
	      identifier { dsnPrefix domain verifiedAt }
            }
          }`

	resp, err := c.DoGQL(ctx, Params{Query: query})
	if err != nil {
		return nil, err
	}

	type response struct {
		Account *Account
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling account: %w", err)
	}

	return data.Account, nil
}
