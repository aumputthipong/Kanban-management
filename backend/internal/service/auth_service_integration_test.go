//go:build integration

// Integration tests for AuthService (Register / Login / UpsertOAuthUser).
//
// AuthService has no test seam of its own — every method hits *db.Queries
// directly, so there's nothing to mock without hitting a real Postgres. Unlike
// a handler test, mocking here would only prove "the mock returns what the
// mock was told to return" — it can't verify the bcrypt hash round-trips, that
// the DB's UNIQUE(email) constraint actually backstops the register race, or
// that ON CONFLICT upsert behaves the way UpsertOAuthUser assumes.
package service_test

import (
	"context"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/testutil"
)

type authFixture struct {
	pool *pgxpool.Pool
	svc  *service.AuthService
}

func newAuthFixture(t *testing.T) *authFixture {
	t.Helper()
	pool := testutil.NewTestDB(t)
	return &authFixture{
		pool: pool,
		svc:  service.NewAuthService(pool, db.New(pool)),
	}
}

// ────────────────────────────────────────────────
// Register
// ────────────────────────────────────────────────

func TestRegister_Success_PasswordIsHashedNotStoredPlain(t *testing.T) {
	ctx := context.Background()
	f := newAuthFixture(t)

	user, err := f.svc.Register(ctx, service.RegisterParams{
		Email: "new@test.local", FullName: "New User", Password: "correct horse battery staple",
	})
	require.NoError(t, err)

	assert.Equal(t, "new@test.local", user.Email)
	assert.Equal(t, "credentials", user.Provider)
	require.NotNil(t, user.PasswordHash)
	// The stored value must be a bcrypt hash, not the raw password — this is
	// the one thing a mock could never verify, since a mock would just hand
	// back whatever string the test told it to.
	assert.NotEqual(t, "correct horse battery staple", *user.PasswordHash)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte("correct horse battery staple")),
		"the stored hash must actually verify against the original password")
}

func TestRegister_DuplicateEmail_ReturnsErrEmailTaken(t *testing.T) {
	ctx := context.Background()
	f := newAuthFixture(t)

	_, err := f.svc.Register(ctx, service.RegisterParams{Email: "dup@test.local", FullName: "First", Password: "pw123456"})
	require.NoError(t, err)

	_, err = f.svc.Register(ctx, service.RegisterParams{Email: "dup@test.local", FullName: "Second", Password: "pw123456"})
	assert.ErrorIs(t, err, service.ErrEmailTaken)
}

// Register does GetUserByEmail then CreateUser as two separate statements —
// the same read-then-write shape as the member/invite races the earlier audit
// flagged (T4). This test doesn't assert the outcome is "correct" (the app
// layer's ErrEmailTaken check can't win this race); it documents what actually
// happens: the database's own UNIQUE(email) constraint is the real backstop,
// not the code. Exactly one registration must succeed either way — that
// invariant is what must never break.
func TestRegister_ConcurrentSameEmail_ExactlyOneSucceeds(t *testing.T) {
	ctx := context.Background()
	f := newAuthFixture(t)

	const goroutines = 8
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results = make([]error, 0, goroutines)
	)
	start := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := f.svc.Register(ctx, service.RegisterParams{
				Email: "racer@test.local", FullName: "Racer", Password: "pw123456",
			})
			mu.Lock()
			results = append(results, err)
			mu.Unlock()
		}()
	}
	close(start)
	wg.Wait()

	successes := 0
	for _, err := range results {
		if err == nil {
			successes++
		}
	}
	assert.Equal(t, 1, successes, "exactly one concurrent registration of the same email may succeed")

	var count int
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE email = 'racer@test.local'`,
	).Scan(&count))
	assert.Equal(t, 1, count, "the DB's UNIQUE(email) constraint must leave exactly one row, not one per racer that read before the first insert")
}

// ────────────────────────────────────────────────
// Login
// ────────────────────────────────────────────────

func TestLogin_CorrectCredentials_Success(t *testing.T) {
	ctx := context.Background()
	f := newAuthFixture(t)
	_, err := f.svc.Register(ctx, service.RegisterParams{Email: "login@test.local", FullName: "Login User", Password: "correct-password"})
	require.NoError(t, err)

	user, err := f.svc.Login(ctx, "login@test.local", "correct-password")
	require.NoError(t, err)
	assert.Equal(t, "login@test.local", user.Email)
}

func TestLogin_WrongPassword_ReturnsErrInvalidCreds(t *testing.T) {
	ctx := context.Background()
	f := newAuthFixture(t)
	_, err := f.svc.Register(ctx, service.RegisterParams{Email: "wrongpw@test.local", FullName: "User", Password: "correct-password"})
	require.NoError(t, err)

	_, err = f.svc.Login(ctx, "wrongpw@test.local", "wrong-password")
	assert.ErrorIs(t, err, service.ErrInvalidCreds)
}

// Unknown email must return the exact same sentinel as a wrong password —
// this is a deliberate security property, not an oversight. If "unknown
// email" and "wrong password" produced different errors, an attacker could
// use the login endpoint to enumerate which emails have accounts.
func TestLogin_UnknownEmail_ReturnsSameErrorAsWrongPassword_NoEnumeration(t *testing.T) {
	ctx := context.Background()
	f := newAuthFixture(t)

	_, err := f.svc.Login(ctx, "nobody-registered@test.local", "anything")
	assert.ErrorIs(t, err, service.ErrInvalidCreds)
}

func TestLogin_OAuthOnlyAccount_ReturnsErrOAuthOnly(t *testing.T) {
	ctx := context.Background()
	f := newAuthFixture(t)
	_, err := f.svc.UpsertOAuthUser(ctx, "oauth@test.local", "OAuth User", "google", "google-sub-123")
	require.NoError(t, err)

	_, err = f.svc.Login(ctx, "oauth@test.local", "any-password")
	assert.ErrorIs(t, err, service.ErrOAuthOnly)
}

// ────────────────────────────────────────────────
// UpsertOAuthUser
// ────────────────────────────────────────────────

func TestUpsertOAuthUser_FirstLogin_CreatesUser(t *testing.T) {
	ctx := context.Background()
	f := newAuthFixture(t)

	user, err := f.svc.UpsertOAuthUser(ctx, "newoauth@test.local", "New OAuth", "google", "sub-1")
	require.NoError(t, err)
	assert.Equal(t, "newoauth@test.local", user.Email)
	assert.Equal(t, "google", user.Provider)
}

// The underlying query is INSERT ... ON CONFLICT (email) DO UPDATE, so a
// returning user must update the same row (same id, refreshed full_name),
// not insert a second one under the same email — the DB's UNIQUE(email)
// would reject a plain duplicate insert, but ON CONFLICT is what makes the
// *second login* succeed instead of erroring.
func TestUpsertOAuthUser_ReturningUser_UpdatesSameRowNotDuplicate(t *testing.T) {
	ctx := context.Background()
	f := newAuthFixture(t)

	first, err := f.svc.UpsertOAuthUser(ctx, "returning@test.local", "Old Name", "google", "sub-2")
	require.NoError(t, err)

	second, err := f.svc.UpsertOAuthUser(ctx, "returning@test.local", "New Name", "google", "sub-2")
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID, "second login must return the same user id, not create a new account")
	assert.Equal(t, "New Name", second.FullName, "full_name must refresh from the latest OAuth profile")

	var count int
	require.NoError(t, f.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE email = 'returning@test.local'`,
	).Scan(&count))
	assert.Equal(t, 1, count)
}

// ────────────────────────────────────────────────
// GetUserByID
// ────────────────────────────────────────────────

func TestGetUserByID_Success(t *testing.T) {
	ctx := context.Background()
	f := newAuthFixture(t)
	created, err := f.svc.Register(ctx, service.RegisterParams{Email: "byid@test.local", FullName: "By ID", Password: "pw123456"})
	require.NoError(t, err)

	found, err := f.svc.GetUserByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "byid@test.local", found.Email)
}
