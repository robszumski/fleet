package api

import (
	"fmt"
	"net/http"
	"path"

	log "github.com/coreos/fleet/Godeps/_workspace/src/github.com/golang/glog"

	"github.com/coreos/fleet/client"
	"github.com/coreos/fleet/machine"
	"github.com/coreos/fleet/schema"
)

func wireUpMachinesResource(mux *http.ServeMux, prefix string, cAPI client.API) {
	res := path.Join(prefix, "machines")
	mr := machinesResource{cAPI}
	mux.Handle(res, &mr)
}

type machinesResource struct {
	cAPI client.API
}

func (mr *machinesResource) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		sendError(rw, http.StatusBadRequest, fmt.Errorf("only HTTP GET supported against this resource"))
		return
	}

	token, err := findNextPageToken(req.URL)
	if err != nil {
		sendError(rw, http.StatusBadRequest, err)
		return
	}

	if token == nil {
		def := DefaultPageToken()
		token = &def
	}

	page, err := getMachinePage(mr.cAPI, *token)
	if err != nil {
		log.Errorf("Failed fetching page of Machines: %v", err)
		sendError(rw, http.StatusInternalServerError, nil)
		return
	}

	sendResponse(rw, http.StatusOK, page)
}

func getMachinePage(cAPI client.API, tok PageToken) (*schema.MachinePage, error) {
	all, err := cAPI.Machines()
	if err != nil {
		return nil, err
	}

	page := extractMachinePage(all, tok)
	return page, nil
}

func extractMachinePage(all []machine.MachineState, tok PageToken) *schema.MachinePage {
	total := len(all)

	startIndex := int((tok.Page - 1) * tok.Limit)
	stopIndex := int(tok.Page * tok.Limit)

	var items []machine.MachineState
	var next *PageToken

	if startIndex < total {
		if stopIndex > total {
			stopIndex = total
		} else {
			n := tok.Next()
			next = &n
		}

		items = all[startIndex:stopIndex]
	}

	return newMachinePage(items, next)
}

func newMachinePage(items []machine.MachineState, tok *PageToken) *schema.MachinePage {
	smp := schema.MachinePage{
		Machines: make([]*schema.Machine, 0, len(items)),
	}

	if tok != nil {
		smp.NextPageToken = tok.Encode()
	}

	for _, sm := range items {
		smp.Machines = append(smp.Machines, schema.MapMachineStateToSchema(&sm))
	}
	return &smp
}
