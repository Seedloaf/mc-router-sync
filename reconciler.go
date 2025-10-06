package mcroutersync

import "fmt"

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
}

type ReconcilerDiff struct {
	ServerAddress  string
	DesiredBackend string // Backend from server list (desired state)
	CurrentBackend string // Backend from mc router (current state)
	InServerList   bool   // Present in server list
	InMcRouter     bool   // Present in mc router
}

type ActionType string

const (
	ActionAdd    ActionType = "add"
	ActionDelete ActionType = "delete"
)

type Action struct {
	Type          ActionType
	ServerAddress string
	Backend       string // Only relevant for Add and Update actions
}

func (r *Reconciler) Diff() ([]ReconcilerDiff, error) {
	// Fetch desired state from server list
	serverListRoutes, err := r.ServerListClient.GetServers()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch server list: %w", err)
	}

	// Fetch current state from mc router
	mcRouterRoutes, err := r.McRouterClient.GetRoutes()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch mc router routes: %w", err)
	}

	// Build maps for easy lookup
	serverListMap := make(map[string]string)
	for _, route := range serverListRoutes {
		serverListMap[route.ServerAddress] = route.Backend
	}

	mcRouterMap := make(map[string]string)
	for _, route := range mcRouterRoutes {
		mcRouterMap[route.ServerAddress] = route.Backend
	}

	// Find all unique server addresses
	allAddresses := make(map[string]bool)
	for addr := range serverListMap {
		allAddresses[addr] = true
	}
	for addr := range mcRouterMap {
		allAddresses[addr] = true
	}

	// Build diff
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
		// Add: route only in server list OR in both but backends differ (update via add)
		if (diff.InServerList && !diff.InMcRouter) || (diff.InServerList && diff.InMcRouter && diff.DesiredBackend != diff.CurrentBackend) {
			actions = append(actions, Action{
				Type:          ActionAdd,
				ServerAddress: diff.ServerAddress,
				Backend:       diff.DesiredBackend,
			})
		} else if !diff.InServerList && diff.InMcRouter {
			// Delete: route only in mc router
			actions = append(actions, Action{
				Type:          ActionDelete,
				ServerAddress: diff.ServerAddress,
			})
		}
		// No action: in both and backends match (skip)
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

func NewReconciler(sl ServerList, mr McRouter) *Reconciler {
	return &Reconciler{
		ServerListClient: sl,
		McRouterClient:   mr,
	}
}
