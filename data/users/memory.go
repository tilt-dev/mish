package users

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/markbates/goth"
	"github.com/windmilleng/mish/data"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type gothKey struct {
	Provider       string
	ProviderUserID string
}

type memoryUserStore struct {
	usersByUserID   map[data.UserID]*data.User
	usersByUsername map[data.Username]*data.User
	usersByGoth     map[gothKey]*data.User
	okGithubLogins  map[string]bool
	lastID          data.UserID
	mu              sync.RWMutex
}

func NewMemoryUserStore(ctx context.Context) (*memoryUserStore, error) {
	store := &memoryUserStore{
		usersByUserID:   make(map[data.UserID]*data.User),
		usersByUsername: make(map[data.Username]*data.User),
		usersByGoth:     make(map[gothKey]*data.User),
		okGithubLogins:  make(map[string]bool),
		lastID:          data.UserID(3),
	}

	err := initStore(ctx, store)
	if err != nil {
		return nil, err
	}
	return store, nil
}

func (s *memoryUserStore) WhitelistedGithubLogins(ctx context.Context) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]string, 0, len(s.okGithubLogins))
	for login := range s.okGithubLogins {
		result = append(result, login)
	}
	return result, nil
}

func (s *memoryUserStore) WhitelistGithubLogin(ctx context.Context, login string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.okGithubLogins[strings.ToLower(login)] = true
	return nil
}

func (s *memoryUserStore) LookupByUserID(ctx context.Context, userID data.UserID) (data.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.usersByUserID[userID]
	if !ok {
		return data.User{}, grpc.Errorf(codes.NotFound, "User not found: %d", userID)
	}
	return *user, nil
}

func (s *memoryUserStore) DeleteByUserID(ctx context.Context, userID data.UserID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, err := s.usersByUserID[userID]
	if !err {
		return grpc.Errorf(codes.NotFound, "User not found: %d", userID)
	}

	for k, v := range s.usersByGoth {
		if v.UserID == userID {
			delete(s.usersByGoth, k)
		}
	}
	delete(s.usersByUsername, user.Username)
	delete(s.usersByUserID, user.UserID)

	return nil
}

func (s *memoryUserStore) LookupByUsername(ctx context.Context, username data.Username) (data.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.usersByUsername[username]
	if !ok {
		return data.User{}, grpc.Errorf(codes.NotFound, "User not found: %s", username)
	}
	return *user, nil
}

func (s *memoryUserStore) LookupByGothUser(ctx context.Context, gUser goth.User) (data.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if gUser.Provider == "" || gUser.UserID == "" {
		return data.User{}, grpc.Errorf(codes.InvalidArgument, "User data invalid: %+v", gUser)
	}

	key := gothKey{Provider: gUser.Provider, ProviderUserID: gUser.UserID}
	user, ok := s.usersByGoth[key]
	if !ok {
		return data.User{}, grpc.Errorf(codes.NotFound, "User not found: (%s, %s)", gUser.Provider, gUser.Name)
	}
	return *user, nil
}

func (s *memoryUserStore) CreateFromGothUser(ctx context.Context, username data.Username, gUser goth.User) (data.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	username = data.NormalizeUsername(username)
	if gUser.Provider == "" || gUser.UserID == "" || username == "" {
		return data.User{}, grpc.Errorf(codes.InvalidArgument, "User data invalid: %+v", gUser)
	}

	if gUser.Provider != "github" && gUser.Provider != "windmill" {
		return data.User{}, grpc.Errorf(codes.InvalidArgument, "Only github accounts allowed: %s", gUser.Provider)
	}

	if gUser.Provider == "github" {
		login := strings.ToLower(fmt.Sprintf("%s", gUser.RawData["login"]))
		_, isWhitelisted := s.okGithubLogins[login]
		if !isWhitelisted {
			return data.User{}, grpc.Errorf(codes.PermissionDenied,
				"Account not whitelisted. Please email nick@windmill.engineering to request access")
		}
	}

	_, existing := s.usersByUsername[username]
	if existing {
		return data.User{}, grpc.Errorf(codes.AlreadyExists, "User already exists: %s", username)
	}

	key := gothKey{Provider: gUser.Provider, ProviderUserID: gUser.UserID}
	existingUser, gExisting := s.usersByGoth[key]
	if gExisting {
		return data.User{},
			grpc.Errorf(codes.AlreadyExists, "User (%s, %s) already linked to Windmill User %s",
				gUser.Provider, gUser.Name, existingUser.Username)
	}

	s.lastID++
	id := s.lastID
	if username == data.Root {
		id = data.RootID
	} else if username == data.Anonymous {
		id = data.AnonymousID
	} else if username == data.Public {
		id = data.PublicID
	}

	user := &data.User{
		UserID:   id,
		Username: username,
		Email:    gUser.Email,
	}
	s.usersByGoth[key] = user
	s.usersByUsername[username] = user
	s.usersByUserID[id] = user
	return *user, nil
}

func (s *memoryUserStore) SetPublicForDemos(ctx context.Context, userID data.UserID, val bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, err := s.usersByUserID[userID]
	if !err {
		return grpc.Errorf(codes.NotFound, "User not found: %d", userID)
	}

	user.IsPublic = val
	return nil
}

var _ data.UserStore = &memoryUserStore{}
