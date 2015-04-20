package send

import (
	"chinchilla/mssg"
	"chinchilla/schedule"
	"chinchilla/types"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Thread to send responses back to hosts
func Client(RespQueue chan mssg.WorkResp, jobs *types.MapJ) {
	for {
		resp := <-RespQueue
		json_resp, _ := json.Marshal(resp)
		jobs.L.Lock()
		w := jobs.M[resp.WId].W
		// allow cross domain AJAX requests
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(200)

		_, err := w.Write(json_resp)
		jobs.L.Unlock()
		close(jobs.M[resp.WId].Sem)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
			os.Exit(1)
		}

	}
}

func Scheduler(w http.ResponseWriter, ReqQueue chan mssg.WorkReq, typ int, arg1 string, id uint32, jobs *types.MapJ) {
	jobs.L.Lock()
	jobs.M[id] = types.Job{W: w, Sem: make(chan struct{})}
	jobs.L.Unlock()
	ReqQueue <- mssg.WorkReq{Type: uint8(typ), Arg1: arg1, WId: id}

}

func Node(ReqQueue chan mssg.WorkReq, workers *types.MapQ) {
	for {
		req := <-ReqQueue
		req.STime = time.Now()
		// node := ShortestQ(workers, req.Type)
		node := schedule.RoundRobin(workers)
		workers.L.Lock()
		tmp := workers.M[node]
		tmp.Reqs = append(workers.M[node].Reqs, req)
		workers.M[node] = tmp
		err := workers.M[node].Enc.Encode(req)
		workers.L.Unlock()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		}
	}
}
