/*
Copyright 2020 IBM Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package listeners

import (
    "github.com/kabanero-io/events-operator/pkg/eventenv"
	"encoding/json"
	"io/ioutil"
	"k8s.io/klog"
	"net/http"
	"os"
    "strconv"
    "sync"
    "fmt"
)

const (
	// HEADER Message key containing request headers
	HEADER = "header"
	// BODY Message key containing request payload
	BODY = "body"
	// WEBHOOKDESTINATION GitHub event destination
	WEBHOOKDESTINATION = "github"

    DEFAULT_PORT = 9443
)


type listenerInfo struct {
    port int
    key string
    handler eventenv.ListenerHandler
    env *eventenv.EventEnv
}

type ListenerManagerDefault struct {
    listeners map[int] *listenerInfo
    mutex sync.Mutex 
}

func NewDefaultListenerManager() eventenv.ListenerManager {
    return &ListenerManagerDefault {
        listeners: make(map[int]*listenerInfo),
    }
}

func (listenerMgr *ListenerManagerDefault) addListener(port int, listener *listenerInfo) error {
	listenerMgr.mutex.Lock()
	defer listenerMgr.mutex.Unlock()

	if _, exists := listenerMgr.listeners[port] ; exists {
		return fmt.Errorf("listener on port %v already exists", port)
	}

	listenerMgr.listeners[port] = listener
	return nil
}

// NewListener creates a new event listener on port 9080
func (listenerMgr *ListenerManagerDefault) NewListener(env *eventenv.EventEnv, port int, key string, handler eventenv.ListenerHandler) error {
	klog.Infof("Starting listener on port %v", port)

	listener := &listenerInfo {
		port: port,
		key: key,
		handler: handler,
		env: env,
	}

	if err := listenerMgr.addListener(port, listener); err != nil {
		return err
	}

	return http.ListenAndServe(":"+ strconv.Itoa(port), listenerHandler(listener))
}


func (listenerMgr *ListenerManagerDefault ) NewListenerTLS(env *eventenv.EventEnv, port int, key string, tlsCertPath, tlsKeyPath string, handler eventenv.ListenerHandler) error {
	klog.Infof("Starting TLS listener on port %v", port)
	if _, err := os.Stat(tlsCertPath); os.IsNotExist(err) {
		klog.Fatalf("TLS certificate '%s' not found: %v", tlsCertPath, err)
		return err
	}
	if _, err := os.Stat(tlsKeyPath); os.IsNotExist(err) {
		klog.Fatalf("TLS private key '%s' not found: %v", tlsKeyPath, err)
		return err
	}

	listener := &listenerInfo {
		port: port,
		key: key,
		handler: handler,
		env: env,
	}

	if err := listenerMgr.addListener(port, listener); err != nil {
		return err
	}

	return http.ListenAndServeTLS(":"+strconv.Itoa(port), tlsCertPath, tlsKeyPath, listenerHandler(listener))
}


/* Event listener */
func listenerHandler(listener *listenerInfo) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {

		header := req.Header
		klog.Infof("Received request. Header: %v", header)

		var body = req.Body
		if body == nil {
			writer.WriteHeader(http.StatusBadRequest)
			klog.Errorf("request does not have a body")
			return
		}
		defer body.Close()

		bytes, err := ioutil.ReadAll(body)
		if err != nil {
			klog.Errorf("listener can not read body. Error: %v", err)
			return
		}

		klog.Infof("listener received body: %v", string(bytes))

		var bodyMap map[string]interface{}
		err = json.Unmarshal(bytes, &bodyMap)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			klog.Errorf("Unable to unmarshal json body: %v", err)
			return
		}

		message := make(map[string]interface{})
		message[HEADER] = map[string][]string(header)
		message[BODY] = bodyMap

		bytes, err = json.Marshal(message)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			klog.Errorf("Unable to marshall as JSON: %v, type %T", message, message)
			return
		}

		err = (listener.handler)(listener.env, message, listener.key, req.URL)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			klog.Errorf("Error processing event: %v", err)
			return
		}
		klog.Info("Completed event processing.\n")

		writer.WriteHeader(http.StatusAccepted)
	}
}

