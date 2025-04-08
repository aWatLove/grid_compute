package tasker_impl

import (
	"encoding/json"
	"log"
	"master-node/internal/config"
)

/*
   "status": "ok",
   "data": {
       "route": best_route,
       "cost": best_cost
   }

*/

type reqSubtask struct {
	Route []int `json:"route"`
	Cost  int   `json:"cost"`
}

type Tasker struct {
	cfg *config.Config

	bestRoute []int
	bestCost  int
}

func New(cfg *config.Config) *Tasker {

	return &Tasker{cfg: cfg}
}

func (t *Tasker) ConfirmSubtaskHandler(message json.RawMessage) {

	var req reqSubtask
	err := json.Unmarshal(message, &req)
	if err != nil {
		log.Printf("[TASKER]Unmarshal error: %v, message: %s", err, string(message))
	}

	log.Printf("[TASKER]ConfirmSubtaskHandler: %v", req)

	if t.bestCost == 0 {
		t.bestCost = req.Cost
		t.bestRoute = req.Route
	}

	if t.bestCost > req.Cost {
		t.bestCost = req.Cost
		t.bestRoute = req.Route
	}

}

func (t *Tasker) DoneTaskHandler() {
	log.Println("[TASKER][EXEC] DoneTaskHandler")
	log.Println("[TASKER][BEST] BestRoute:", t.bestRoute)
	log.Println("[TASKER][BEST] BestCost:", t.bestCost)

}

func (t *Tasker) ErrorTaskHandler(err error) {
	log.Fatalln("[TASKER] Error:", err)

}
