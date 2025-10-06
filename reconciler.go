package mcroutersync

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type ServerList interface {
	GetServers() (Routes, error)
}

type McRouter interface {
	GetRoutes() (Routes, error)
	RegisterRoute(route Route) error
	DeleteRoute(serverAddress string) error
}

type Reconciler struct {
	ServerListClient ServerList
	McRouterClient   McRouter
	Interval         time.Duration
}

type ReconcilerDiff struct {
	ServerAddress  string
	DesiredBackend string
	CurrentBackend string
	InServerList   bool
	InMcRouter     bool
}

type ActionType string

const (
	ActionAdd    ActionType = "add"
	ActionDelete ActionType = "delete"
)

type Action struct {
	Type          ActionType
	ServerAddress string
	Backend       string
}

func (r *Reconciler) Start(ctx context.Context) {
	ticker := time.NewTicker(r.Interval)
	defer ticker.Stop()

	if err := r.Reconcile(); err != nil {
		slog.Error("reconciliation error", "err", err)
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("reconciler stopped")
			return
		case <-ticker.C:
			if err := r.Reconcile(); err != nil {
				slog.Error("reconciliation error", "err", err)
			}
		}
	}
}

func (r *Reconciler) Reconcile() error {
	diffs, err := r.Diff()
	if err != nil {
		return fmt.Errorf("failed to diff: %w", err)
	}
	slog.Debug("Reconciling diffs", "diffs", diffs)

	actions := r.Actions(diffs)
	slog.Debug("Applying Actions", "actions", actions)
	err = r.Apply(actions)
	if err != nil {
		return fmt.Errorf("failed to apply actions: %w", err)
	}

	return nil
}

func (r *Reconciler) Diff() ([]ReconcilerDiff, error) {
	serverListRoutes, err := r.ServerListClient.GetServers()
	if err != nil {
		return nil, fmt.Errorf("failed to get servers: %w", err)
	}

	mcRouterRoutes, err := r.McRouterClient.GetRoutes()
	if err != nil {
		return nil, fmt.Errorf("failed to get routes: %w", err)
	}

	serverListMap := make(map[string]string)
	for _, route := range serverListRoutes {
		serverListMap[route.ServerAddress] = route.Backend
	}

	mcRouterMap := make(map[string]string)
	for _, route := range mcRouterRoutes {
		mcRouterMap[route.ServerAddress] = route.Backend
	}

	allAddresses := make(map[string]bool)
	for addr := range serverListMap {
		allAddresses[addr] = true
	}
	for addr := range mcRouterMap {
		allAddresses[addr] = true
	}

	var diffs []ReconcilerDiff
	for addr := range allAddresses {
		desiredBackend, inServerList := serverListMap[addr]
		currentBackend, inMcRouter := mcRouterMap[addr]

		diffs = append(diffs, ReconcilerDiff{
			ServerAddress:  addr,
			DesiredBackend: desiredBackend,
			CurrentBackend: currentBackend,
			InServerList:   inServerList,
			InMcRouter:     inMcRouter,
		})
	}

	return diffs, nil
}

func (r *Reconciler) Actions(diffs []ReconcilerDiff) []Action {
	var actions []Action

	for _, diff := range diffs {
		if (diff.InServerList && !diff.InMcRouter) || (diff.InServerList && diff.InMcRouter && diff.DesiredBackend != diff.CurrentBackend) {
			actions = append(actions, Action{
				Type:          ActionAdd,
				ServerAddress: diff.ServerAddress,
				Backend:       diff.DesiredBackend,
			})
		} else if !diff.InServerList && diff.InMcRouter {
			actions = append(actions, Action{
				Type:          ActionDelete,
				ServerAddress: diff.ServerAddress,
			})
		}
	}

	return actions
}

func (r *Reconciler) Apply(actions []Action) error {
	for _, action := range actions {
		switch action.Type {
		case ActionAdd:
			route := Route{
				ServerAddress: action.ServerAddress,
				Backend:       action.Backend,
			}
			if err := r.McRouterClient.RegisterRoute(route); err != nil {
				return fmt.Errorf("failed to register route %s: %w", action.ServerAddress, err)
			}
		case ActionDelete:
			if err := r.McRouterClient.DeleteRoute(action.ServerAddress); err != nil {
				return fmt.Errorf("failed to delete route %s: %w", action.ServerAddress, err)
			}
		}
	}
	return nil
}

func NewReconciler(sl ServerList, mr McRouter, interval time.Duration) *Reconciler {
	return &Reconciler{
		ServerListClient: sl,
		McRouterClient:   mr,
		Interval:         interval,
	}
}
