package communicator

import (
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/netly/agent/internal/executor"
)

type AgentServer struct {
	port int
}

func NewAgentServer(port int) *AgentServer {
	return &AgentServer{port: port}
}

func (s *AgentServer) Start() error {
	http.HandleFunc("/api/v1/agent/self", s.handleSelfDestruct)
	
	// Listen on 0.0.0.0 to allow backend connection
	addr := fmt.Sprintf(":%d", s.port)
	return http.ListenAndServe(addr, nil)
}

func (s *AgentServer) handleSelfDestruct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Trigger self-destruct asynchronously
	go func() {
		if err := executor.PerformSelfDestruct(); err != nil {
			fmt.Println("Self destruct failed:", err)
		}
	}()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "self-destruct initiated"})
}
