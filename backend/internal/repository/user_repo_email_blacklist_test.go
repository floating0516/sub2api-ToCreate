//go:build unit

package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestUserRepositoryIsEmailDomainBlacklisted(t *testing.T) {
	tests := []struct {
		name string
		row  bool
		want bool
	}{
		{name: "exact domain", row: true, want: true},
		{name: "included subdomain", row: true, want: true},
		{name: "disabled or unmatched", row: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })

			mock.ExpectQuery(`(?s)SELECT EXISTS \(.*FROM email_domain_blacklist.*RIGHT`).
				WithArgs("mail.12api.buzz").
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(tt.row))

			repo := newUserRepositoryWithSQL(nil, db)
			got, err := repo.IsEmailDomainBlacklisted(context.Background(), " MAIL.12API.BUZZ. ")

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepositoryIsEmailDomainBlacklistedSkipsEmptyDomain(t *testing.T) {
	repo := newUserRepositoryWithSQL(nil, nil)

	got, err := repo.IsEmailDomainBlacklisted(context.Background(), " . ")

	require.NoError(t, err)
	require.False(t, got)
}

func TestIsEmailDomainBlacklistViolation(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", &pq.Error{
		Code:    "23514",
		Message: "EMAIL_DOMAIN_BLACKLISTED: mail.12api.buzz",
	})

	require.True(t, isEmailDomainBlacklistViolation(err))
	require.False(t, isEmailDomainBlacklistViolation(&pq.Error{Code: "23514", Message: "other check failed"}))
}
