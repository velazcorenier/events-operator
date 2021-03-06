package eventenv

import (
	"github.com/kabanero-io/events-operator/pkg/connections"
	"github.com/kabanero-io/events-operator/pkg/listeners"
	"github.com/kabanero-io/events-operator/pkg/managers"
	"github.com/kabanero-io/events-operator/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MEDIATOR_NAME_KEY = "MEDIATOR-NAME" // environment variable. If not set, we're running as operator.
)

type EventEnv struct {
	Client              client.Client
	EventMgr            *managers.EventManager
	ConnectionsMgr      *connections.ConnectionsManager
	ListenerMgr         listeners.ListenerManager
    StatusMgr           *status.StatusManager
    StatusUpdater       *status.Updater
	MediatorName        string // Kubernetes name of this mediator worker if not ""
	IsOperator          bool   // true if this instance is an operator, not a worker
	Namespace           string // namespace we're running under
	KabaneroIntegration bool   // true to integrate with Kabanero
}

var eventEnv *EventEnv

func InitEventEnv(env *EventEnv) {
	eventEnv = env
}

func GetEventEnv() *EventEnv {
	return eventEnv
}
