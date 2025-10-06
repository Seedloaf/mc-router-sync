package mcroutersync

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

type mockServerList struct {
	routes Routes
	err    error
}

func (m *mockServerList) GetServers() (Routes, error) {
	return m.routes, m.err
}

type mockMcRouter struct {
	routes             Routes
	err                error
	deleteErr          error
	registerErr        error
	getRoutesCallCount int
}

func (m *mockMcRouter) GetRoutes() (Routes, error) {
	m.getRoutesCallCount++
	return m.routes, m.err
}

func (m *mockMcRouter) DeleteRoute(serverAddress string) error {
	return m.deleteErr
}

func (m *mockMcRouter) RegisterRoute(route Route) error {
	return m.registerErr
}

func TestNewReconciler(t *testing.T) {
	sl := &mockServerList{}
	mr := &mockMcRouter{}

	reconciler := NewReconciler(sl, mr, 30*time.Second)

	if reconciler == nil {
		t.Fatal("expected reconciler to be non-nil")
	}
	if reconciler.ServerListClient == nil {
		t.Error("expected ServerListClient to be non-nil")
	}
	if reconciler.McRouterClient == nil {
		t.Error("expected McRouterClient to be non-nil")
	}
	if reconciler.Interval != 30*time.Second {
		t.Errorf("expected interval to be 30s, got %v", reconciler.Interval)
	}
}

func TestReconcilerDiff(t *testing.T) {
	tests := []struct {
		name              string
		serverListRoutes  Routes
		mcRouterRoutes    Routes
		serverListError   error
		mcRouterError     error
		expectError       bool
		expectedDiffCount int
		validateDiffs     func(t *testing.T, diffs []ReconcilerDiff)
	}{
		{
			name: "routes in sync",
			serverListRoutes: Routes{
				{ServerAddress: "server1.example.com", Backend: "backend1:25565"},
				{ServerAddress: "server2.example.com", Backend: "backend2:25565"},
			},
			mcRouterRoutes: Routes{
				{ServerAddress: "server1.example.com", Backend: "backend1:25565"},
				{ServerAddress: "server2.example.com", Backend: "backend2:25565"},
			},
			expectError:       false,
			expectedDiffCount: 2,
			validateDiffs: func(t *testing.T, diffs []ReconcilerDiff) {
				for _, diff := range diffs {
					if !diff.InServerList {
						t.Errorf("expected %s to be in server list", diff.ServerAddress)
					}
					if !diff.InMcRouter {
						t.Errorf("expected %s to be in mc router", diff.ServerAddress)
					}
					if diff.DesiredBackend != diff.CurrentBackend {
						t.Errorf("expected backends to match for %s", diff.ServerAddress)
					}
				}
			},
		},
		{
			name: "route needs update",
			serverListRoutes: Routes{
				{ServerAddress: "server1.example.com", Backend: "backend1:25565"},
			},
			mcRouterRoutes: Routes{
				{ServerAddress: "server1.example.com", Backend: "old-backend:25565"},
			},
			expectError:       false,
			expectedDiffCount: 1,
			validateDiffs: func(t *testing.T, diffs []ReconcilerDiff) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				diff := diffs[0]
				if diff.ServerAddress != "server1.example.com" {
					t.Errorf("expected server address server1.example.com, got %s", diff.ServerAddress)
				}
				if diff.DesiredBackend != "backend1:25565" {
					t.Errorf("expected desired backend backend1:25565, got %s", diff.DesiredBackend)
				}
				if diff.CurrentBackend != "old-backend:25565" {
					t.Errorf("expected current backend old-backend:25565, got %s", diff.CurrentBackend)
				}
				if !diff.InServerList || !diff.InMcRouter {
					t.Error("expected route to be in both server list and mc router")
				}
			},
		},
		{
			name: "route only in server list (needs to be added)",
			serverListRoutes: Routes{
				{ServerAddress: "new-server.example.com", Backend: "backend1:25565"},
			},
			mcRouterRoutes:    Routes{},
			expectError:       false,
			expectedDiffCount: 1,
			validateDiffs: func(t *testing.T, diffs []ReconcilerDiff) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				diff := diffs[0]
				if diff.ServerAddress != "new-server.example.com" {
					t.Errorf("expected server address new-server.example.com, got %s", diff.ServerAddress)
				}
				if !diff.InServerList {
					t.Error("expected route to be in server list")
				}
				if diff.InMcRouter {
					t.Error("expected route to not be in mc router")
				}
				if diff.DesiredBackend != "backend1:25565" {
					t.Errorf("expected desired backend backend1:25565, got %s", diff.DesiredBackend)
				}
				if diff.CurrentBackend != "" {
					t.Errorf("expected empty current backend, got %s", diff.CurrentBackend)
				}
			},
		},
		{
			name:             "route only in mc router (needs to be removed)",
			serverListRoutes: Routes{},
			mcRouterRoutes: Routes{
				{ServerAddress: "old-server.example.com", Backend: "backend1:25565"},
			},
			expectError:       false,
			expectedDiffCount: 1,
			validateDiffs: func(t *testing.T, diffs []ReconcilerDiff) {
				if len(diffs) != 1 {
					t.Fatalf("expected 1 diff, got %d", len(diffs))
				}
				diff := diffs[0]
				if diff.ServerAddress != "old-server.example.com" {
					t.Errorf("expected server address old-server.example.com, got %s", diff.ServerAddress)
				}
				if diff.InServerList {
					t.Error("expected route to not be in server list")
				}
				if !diff.InMcRouter {
					t.Error("expected route to be in mc router")
				}
				if diff.DesiredBackend != "" {
					t.Errorf("expected empty desired backend, got %s", diff.DesiredBackend)
				}
				if diff.CurrentBackend != "backend1:25565" {
					t.Errorf("expected current backend backend1:25565, got %s", diff.CurrentBackend)
				}
			},
		},
		{
			name: "complex scenario with multiple differences",
			serverListRoutes: Routes{
				{ServerAddress: "server1.example.com", Backend: "backend1:25565"},
				{ServerAddress: "server2.example.com", Backend: "new-backend:25565"},
				{ServerAddress: "server3.example.com", Backend: "backend3:25565"},
			},
			mcRouterRoutes: Routes{
				{ServerAddress: "server1.example.com", Backend: "backend1:25565"},
				{ServerAddress: "server2.example.com", Backend: "old-backend:25565"},
				{ServerAddress: "server4.example.com", Backend: "backend4:25565"},
			},
			expectError:       false,
			expectedDiffCount: 4,
			validateDiffs: func(t *testing.T, diffs []ReconcilerDiff) {
				diffMap := make(map[string]ReconcilerDiff)
				for _, diff := range diffs {
					diffMap[diff.ServerAddress] = diff
				}

				if diff, ok := diffMap["server1.example.com"]; ok {
					if !diff.InServerList || !diff.InMcRouter {
						t.Error("server1 should be in both")
					}
					if diff.DesiredBackend != diff.CurrentBackend {
						t.Error("server1 backends should match")
					}
				} else {
					t.Error("server1 should be in diffs")
				}

				if diff, ok := diffMap["server2.example.com"]; ok {
					if !diff.InServerList || !diff.InMcRouter {
						t.Error("server2 should be in both")
					}
					if diff.DesiredBackend == diff.CurrentBackend {
						t.Error("server2 backends should not match")
					}
				} else {
					t.Error("server2 should be in diffs")
				}

				if diff, ok := diffMap["server3.example.com"]; ok {
					if !diff.InServerList || diff.InMcRouter {
						t.Error("server3 should only be in server list")
					}
				} else {
					t.Error("server3 should be in diffs")
				}

				if diff, ok := diffMap["server4.example.com"]; ok {
					if diff.InServerList || !diff.InMcRouter {
						t.Error("server4 should only be in mc router")
					}
				} else {
					t.Error("server4 should be in diffs")
				}
			},
		},
		{
			name:              "empty routes on both sides",
			serverListRoutes:  Routes{},
			mcRouterRoutes:    Routes{},
			expectError:       false,
			expectedDiffCount: 0,
		},
		{
			name:             "server list fetch error",
			serverListRoutes: nil,
			mcRouterRoutes:   Routes{},
			serverListError:  errors.New("failed to fetch server list"),
			expectError:      true,
		},
		{
			name:             "mc router fetch error",
			serverListRoutes: Routes{},
			mcRouterRoutes:   nil,
			mcRouterError:    errors.New("failed to fetch mc router routes"),
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sl := &mockServerList{
				routes: tt.serverListRoutes,
				err:    tt.serverListError,
			}
			mr := &mockMcRouter{
				routes: tt.mcRouterRoutes,
				err:    tt.mcRouterError,
			}

			reconciler := NewReconciler(sl, mr, 30*time.Second)
			diffs, err := reconciler.Diff()

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(diffs) != tt.expectedDiffCount {
				t.Errorf("expected %d diffs, got %d", tt.expectedDiffCount, len(diffs))
			}

			if tt.validateDiffs != nil {
				tt.validateDiffs(t, diffs)
			}
		})
	}
}

func TestReconcilerActions(t *testing.T) {
	tests := []struct {
		name            string
		diffs           []ReconcilerDiff
		expectedActions []Action
	}{
		{
			name: "no actions needed - routes in sync",
			diffs: []ReconcilerDiff{
				{
					ServerAddress:  "server1.example.com",
					DesiredBackend: "backend1:25565",
					CurrentBackend: "backend1:25565",
					InServerList:   true,
					InMcRouter:     true,
				},
				{
					ServerAddress:  "server2.example.com",
					DesiredBackend: "backend2:25565",
					CurrentBackend: "backend2:25565",
					InServerList:   true,
					InMcRouter:     true,
				},
			},
			expectedActions: []Action{},
		},
		{
			name: "add action - route only in server list",
			diffs: []ReconcilerDiff{
				{
					ServerAddress:  "new-server.example.com",
					DesiredBackend: "backend1:25565",
					CurrentBackend: "",
					InServerList:   true,
					InMcRouter:     false,
				},
			},
			expectedActions: []Action{
				{
					Type:          ActionAdd,
					ServerAddress: "new-server.example.com",
					Backend:       "backend1:25565",
				},
			},
		},
		{
			name: "delete action - route only in mc router",
			diffs: []ReconcilerDiff{
				{
					ServerAddress:  "old-server.example.com",
					DesiredBackend: "",
					CurrentBackend: "backend1:25565",
					InServerList:   false,
					InMcRouter:     true,
				},
			},
			expectedActions: []Action{
				{
					Type:          ActionDelete,
					ServerAddress: "old-server.example.com",
				},
			},
		},
		{
			name: "add action - route in both but different backends (update via add)",
			diffs: []ReconcilerDiff{
				{
					ServerAddress:  "server1.example.com",
					DesiredBackend: "new-backend:25565",
					CurrentBackend: "old-backend:25565",
					InServerList:   true,
					InMcRouter:     true,
				},
			},
			expectedActions: []Action{
				{
					Type:          ActionAdd,
					ServerAddress: "server1.example.com",
					Backend:       "new-backend:25565",
				},
			},
		},
		{
			name: "mixed actions",
			diffs: []ReconcilerDiff{
				{
					ServerAddress:  "in-sync.example.com",
					DesiredBackend: "backend1:25565",
					CurrentBackend: "backend1:25565",
					InServerList:   true,
					InMcRouter:     true,
				},
				{
					ServerAddress:  "to-add.example.com",
					DesiredBackend: "backend2:25565",
					CurrentBackend: "",
					InServerList:   true,
					InMcRouter:     false,
				},
				{
					ServerAddress:  "to-delete.example.com",
					DesiredBackend: "",
					CurrentBackend: "backend3:25565",
					InServerList:   false,
					InMcRouter:     true,
				},
				{
					ServerAddress:  "to-update.example.com",
					DesiredBackend: "new-backend:25565",
					CurrentBackend: "old-backend:25565",
					InServerList:   true,
					InMcRouter:     true,
				},
			},
			expectedActions: []Action{
				{
					Type:          ActionAdd,
					ServerAddress: "to-add.example.com",
					Backend:       "backend2:25565",
				},
				{
					Type:          ActionDelete,
					ServerAddress: "to-delete.example.com",
				},
				{
					Type:          ActionAdd,
					ServerAddress: "to-update.example.com",
					Backend:       "new-backend:25565",
				},
			},
		},
		{
			name:            "empty diffs",
			diffs:           []ReconcilerDiff{},
			expectedActions: []Action{},
		},
		{
			name: "multiple adds",
			diffs: []ReconcilerDiff{
				{
					ServerAddress:  "server1.example.com",
					DesiredBackend: "backend1:25565",
					CurrentBackend: "",
					InServerList:   true,
					InMcRouter:     false,
				},
				{
					ServerAddress:  "server2.example.com",
					DesiredBackend: "backend2:25565",
					CurrentBackend: "",
					InServerList:   true,
					InMcRouter:     false,
				},
			},
			expectedActions: []Action{
				{
					Type:          ActionAdd,
					ServerAddress: "server1.example.com",
					Backend:       "backend1:25565",
				},
				{
					Type:          ActionAdd,
					ServerAddress: "server2.example.com",
					Backend:       "backend2:25565",
				},
			},
		},
		{
			name: "multiple deletes",
			diffs: []ReconcilerDiff{
				{
					ServerAddress:  "server1.example.com",
					DesiredBackend: "",
					CurrentBackend: "backend1:25565",
					InServerList:   false,
					InMcRouter:     true,
				},
				{
					ServerAddress:  "server2.example.com",
					DesiredBackend: "",
					CurrentBackend: "backend2:25565",
					InServerList:   false,
					InMcRouter:     true,
				},
			},
			expectedActions: []Action{
				{
					Type:          ActionDelete,
					ServerAddress: "server1.example.com",
				},
				{
					Type:          ActionDelete,
					ServerAddress: "server2.example.com",
				},
			},
		},
		{
			name: "multiple updates (via add)",
			diffs: []ReconcilerDiff{
				{
					ServerAddress:  "server1.example.com",
					DesiredBackend: "new-backend1:25565",
					CurrentBackend: "old-backend1:25565",
					InServerList:   true,
					InMcRouter:     true,
				},
				{
					ServerAddress:  "server2.example.com",
					DesiredBackend: "new-backend2:25565",
					CurrentBackend: "old-backend2:25565",
					InServerList:   true,
					InMcRouter:     true,
				},
			},
			expectedActions: []Action{
				{
					Type:          ActionAdd,
					ServerAddress: "server1.example.com",
					Backend:       "new-backend1:25565",
				},
				{
					Type:          ActionAdd,
					ServerAddress: "server2.example.com",
					Backend:       "new-backend2:25565",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reconciler := &Reconciler{}
			actions := reconciler.Actions(tt.diffs)

			if len(actions) != len(tt.expectedActions) {
				t.Errorf("expected %d actions, got %d", len(tt.expectedActions), len(actions))
				return
			}

			expectedMap := make(map[string]Action)
			for _, action := range tt.expectedActions {
				key := string(action.Type) + ":" + action.ServerAddress
				expectedMap[key] = action
			}

			for _, action := range actions {
				key := string(action.Type) + ":" + action.ServerAddress
				expected, found := expectedMap[key]
				if !found {
					t.Errorf("unexpected action: %+v", action)
					continue
				}

				if action.Type != expected.Type {
					t.Errorf("expected action type %s, got %s", expected.Type, action.Type)
				}
				if action.ServerAddress != expected.ServerAddress {
					t.Errorf("expected server address %s, got %s", expected.ServerAddress, action.ServerAddress)
				}
				if action.Backend != expected.Backend {
					t.Errorf("expected backend %s, got %s", expected.Backend, action.Backend)
				}
			}
		})
	}
}

func TestReconcilerApply(t *testing.T) {
	tests := []struct {
		name        string
		actions     []Action
		registerErr error
		deleteErr   error
		expectError bool
		errorMsg    string
	}{
		{
			name:        "no actions",
			actions:     []Action{},
			expectError: false,
		},
		{
			name: "successful add action",
			actions: []Action{
				{
					Type:          ActionAdd,
					ServerAddress: "server1.example.com",
					Backend:       "backend1:25565",
				},
			},
			expectError: false,
		},
		{
			name: "successful delete action",
			actions: []Action{
				{
					Type:          ActionDelete,
					ServerAddress: "server1.example.com",
				},
			},
			expectError: false,
		},
		{
			name: "successful mixed actions",
			actions: []Action{
				{
					Type:          ActionAdd,
					ServerAddress: "server1.example.com",
					Backend:       "backend1:25565",
				},
				{
					Type:          ActionAdd,
					ServerAddress: "server2.example.com",
					Backend:       "backend2:25565",
				},
				{
					Type:          ActionDelete,
					ServerAddress: "server3.example.com",
				},
			},
			expectError: false,
		},
		{
			name: "register route fails",
			actions: []Action{
				{
					Type:          ActionAdd,
					ServerAddress: "server1.example.com",
					Backend:       "backend1:25565",
				},
			},
			registerErr: fmt.Errorf("failed to register"),
			expectError: true,
			errorMsg:    "failed to register route",
		},
		{
			name: "delete route fails",
			actions: []Action{
				{
					Type:          ActionDelete,
					ServerAddress: "server1.example.com",
				},
			},
			deleteErr:   fmt.Errorf("failed to delete"),
			expectError: true,
			errorMsg:    "failed to delete route",
		},
		{
			name: "fails on first error in mixed actions",
			actions: []Action{
				{
					Type:          ActionAdd,
					ServerAddress: "server1.example.com",
					Backend:       "backend1:25565",
				},
				{
					Type:          ActionDelete,
					ServerAddress: "server2.example.com",
				},
			},
			registerErr: fmt.Errorf("failed to register"),
			expectError: true,
			errorMsg:    "failed to register route",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := &mockMcRouter{
				registerErr: tt.registerErr,
				deleteErr:   tt.deleteErr,
			}
			sl := &mockServerList{}

			reconciler := NewReconciler(sl, mr, 30*time.Second)
			err := reconciler.Apply(tt.actions)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr)+1 && findSubstr(s, substr)))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestReconcilerReconcile(t *testing.T) {
	tests := []struct {
		name             string
		serverListRoutes Routes
		mcRouterRoutes   Routes
		serverListErr    error
		mcRouterErr      error
		registerErr      error
		deleteErr        error
		expectError      bool
	}{
		{
			name: "successful reconciliation - add route",
			serverListRoutes: Routes{
				{ServerAddress: "server1.example.com", Backend: "backend1:25565"},
			},
			mcRouterRoutes: Routes{},
			expectError:    false,
		},
		{
			name:             "successful reconciliation - delete route",
			serverListRoutes: Routes{},
			mcRouterRoutes: Routes{
				{ServerAddress: "server1.example.com", Backend: "backend1:25565"},
			},
			expectError: false,
		},
		{
			name: "successful reconciliation - update route",
			serverListRoutes: Routes{
				{ServerAddress: "server1.example.com", Backend: "new-backend:25565"},
			},
			mcRouterRoutes: Routes{
				{ServerAddress: "server1.example.com", Backend: "old-backend:25565"},
			},
			expectError: false,
		},
		{
			name: "successful reconciliation - no changes needed",
			serverListRoutes: Routes{
				{ServerAddress: "server1.example.com", Backend: "backend1:25565"},
			},
			mcRouterRoutes: Routes{
				{ServerAddress: "server1.example.com", Backend: "backend1:25565"},
			},
			expectError: false,
		},
		{
			name:             "diff error - server list fetch fails",
			serverListRoutes: Routes{},
			serverListErr:    fmt.Errorf("failed to fetch"),
			expectError:      true,
		},
		{
			name:             "diff error - mc router fetch fails",
			serverListRoutes: Routes{},
			mcRouterErr:      fmt.Errorf("failed to fetch"),
			expectError:      true,
		},
		{
			name: "apply error - register fails",
			serverListRoutes: Routes{
				{ServerAddress: "server1.example.com", Backend: "backend1:25565"},
			},
			mcRouterRoutes: Routes{},
			registerErr:    fmt.Errorf("failed to register"),
			expectError:    true,
		},
		{
			name:             "apply error - delete fails",
			serverListRoutes: Routes{},
			mcRouterRoutes: Routes{
				{ServerAddress: "server1.example.com", Backend: "backend1:25565"},
			},
			deleteErr:   fmt.Errorf("failed to delete"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sl := &mockServerList{
				routes: tt.serverListRoutes,
				err:    tt.serverListErr,
			}
			mr := &mockMcRouter{
				routes:      tt.mcRouterRoutes,
				err:         tt.mcRouterErr,
				registerErr: tt.registerErr,
				deleteErr:   tt.deleteErr,
			}

			reconciler := NewReconciler(sl, mr, 30*time.Second)
			err := reconciler.Reconcile()

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestReconcilerStart(t *testing.T) {
	t.Run("stops on context cancellation", func(t *testing.T) {
		sl := &mockServerList{
			routes: Routes{},
		}
		mr := &mockMcRouter{
			routes: Routes{},
		}

		reconciler := NewReconciler(sl, mr, 100*time.Millisecond)

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan struct{})
		go func() {
			reconciler.Start(ctx)
			close(done)
		}()

		time.Sleep(250 * time.Millisecond)

		cancel()

		select {
		case <-done:

		case <-time.After(1 * time.Second):
			t.Fatal("Start did not stop after context cancellation")
		}
	})

	t.Run("continues on reconciliation error", func(t *testing.T) {
		sl := &mockServerList{
			routes: Routes{},
		}
		mr := &mockMcRouter{
			routes: Routes{},
			err:    fmt.Errorf("simulated error"),
		}

		reconciler := NewReconciler(sl, mr, 50*time.Millisecond)

		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		defer cancel()

		reconciler.Start(ctx)

		if mr.getRoutesCallCount < 2 {
			t.Errorf("expected at least 2 reconciliation attempts, got %d", mr.getRoutesCallCount)
		}
	})
}
