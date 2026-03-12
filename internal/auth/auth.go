package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	db        *sql.DB
	jwtSecret []byte
	limiter   *loginLimiter
}

type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

type Claims struct {
	UserID int64 `json:"userId"`
	jwt.RegisteredClaims
}

func NewService(db *sql.DB, secret string) *Service {
	return &Service{
		db:        db,
		jwtSecret: []byte(secret),
		limiter:   newLoginLimiter(5, 15*time.Minute),
	}
}

func (s *Service) Register(ctx context.Context, email, password string) (User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.Contains(email, "@") {
		return User{}, "", errors.New("valid email is required")
	}
	if len(password) < 8 {
		return User{}, "", errors.New("password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, "", fmt.Errorf("hash password: %w", err)
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO users (email, password_hash) VALUES (?, ?)
	`, email, string(hash))
	if err != nil {
		return User{}, "", fmt.Errorf("create user: %w", err)
	}

	id, _ := result.LastInsertId()
	user := User{ID: id, Email: email, CreatedAt: time.Now().UTC()}
	token, err := s.issueToken(user.ID)
	return user, token, err
}

func (s *Service) Login(ctx context.Context, email, password, limiterKey string) (User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return User{}, "", errors.New("email and password are required")
	}
	if limiterKey == "" {
		limiterKey = email
	}
	if !s.limiter.Allow(limiterKey) {
		return User{}, "", errors.New("too many failed login attempts, try again later")
	}

	var user User
	var passwordHash string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, created_at
		FROM users
		WHERE email = ?
	`, email).Scan(&user.ID, &user.Email, &passwordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.limiter.RecordFailure(limiterKey)
			return User{}, "", errors.New("invalid credentials")
		}
		return User{}, "", fmt.Errorf("lookup user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		s.limiter.RecordFailure(limiterKey)
		return User{}, "", errors.New("invalid credentials")
	}

	s.limiter.Reset(limiterKey)
	token, err := s.issueToken(user.ID)
	return user, token, err
}

func (s *Service) GetUser(ctx context.Context, id int64) (User, error) {
	var user User
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, created_at
		FROM users
		WHERE id = ?
	`, id).Scan(&user.ID, &user.Email, &user.CreatedAt)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *Service) VerifyPassword(ctx context.Context, userID int64, password string) error {
	var passwordHash string
	err := s.db.QueryRowContext(ctx, `
		SELECT password_hash
		FROM users
		WHERE id = ?
	`, userID).Scan(&passwordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("user not found")
		}
		return fmt.Errorf("lookup password hash: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return errors.New("invalid credentials")
	}
	return nil
}

func (s *Service) issueToken(userID int64) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(12 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})

	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}

type contextKey string

const UserIDKey contextKey = "userID"

func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(header, "Bearer ")
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return s.jwtSecret, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			http.Error(w, "invalid token claims", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) int64 {
	id, _ := ctx.Value(UserIDKey).(int64)
	return id
}

type loginLimiter struct {
	mu          sync.Mutex
	maxAttempts int
	window      time.Duration
	attempts    map[string]loginAttempt
}

type loginAttempt struct {
	count       int
	blockedUntil time.Time
}

func newLoginLimiter(maxAttempts int, window time.Duration) *loginLimiter {
	return &loginLimiter{
		maxAttempts: maxAttempts,
		window:      window,
		attempts:    make(map[string]loginAttempt),
	}
}

func (l *loginLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.attempts[key]
	if !ok {
		return true
	}
	if !entry.blockedUntil.IsZero() && time.Now().Before(entry.blockedUntil) {
		return false
	}
	if !entry.blockedUntil.IsZero() && time.Now().After(entry.blockedUntil) {
		delete(l.attempts, key)
	}
	return true
}

func (l *loginLimiter) RecordFailure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := l.attempts[key]
	entry.count++
	if entry.count >= l.maxAttempts {
		entry.blockedUntil = time.Now().Add(l.window)
		entry.count = 0
	}
	l.attempts[key] = entry
}

func (l *loginLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}
