package app

import (
	"net/http"
	"os"

	"github.com/jaegertracing/jaeger/model"
	ui "github.com/jaegertracing/jaeger/model/json"
	"github.com/jaegertracing/jaeger/storage/spanstore"
	"go.uber.org/zap"
)

//environment variable
const (
	envTenantCode      = "JAGER_TENANT_CODE"       //tenant
	envAdminTenantCode = "JAGER_ADMIN_TENANT_CODE" //admin tenant, admin can visit all trace.
	envTenantTag       = "JAGER_TENANT_TAG"        //tenant tag for search
)

func (aH *APIHandler) searchForMultitenancy(w http.ResponseWriter, r *http.Request) {
	tQuery, err := aH.queryParser.parse(r)
	if aH.handleError(w, err, http.StatusBadRequest) {
		return
	}
	//for multitenancy
	if os.Getenv(envTenantCode) != "" && os.Getenv(envTenantTag) != "" {
		searchTag := os.Getenv(envTenantTag)
		tenantCode := r.Header.Get(os.Getenv(envTenantCode))
		adminTenantCode := os.Getenv(envAdminTenantCode)

		if tenantCode != "" && tenantCode != adminTenantCode {
			aH.logger.Info("search by tenant", zap.String("tenant", tenantCode))
			tQuery.Tags[searchTag] = tenantCode
		}
	}

	var uiErrors []structuredError
	var tracesFromStorage []*model.Trace
	if len(tQuery.traceIDs) > 0 {
		tracesFromStorage, uiErrors, err = aH.tracesByIDs(r.Context(), tQuery.traceIDs)
		if aH.handleError(w, err, http.StatusInternalServerError) {
			return
		}
	} else {
		tracesFromStorage, err = aH.queryService.FindTraces(r.Context(), &tQuery.TraceQueryParameters)
		if aH.handleError(w, err, http.StatusInternalServerError) {
			return
		}
	}

	uiTraces := make([]*ui.Trace, len(tracesFromStorage))
	for i, v := range tracesFromStorage {
		uiTrace, uiErr := aH.convertModelToUI(v, true)
		if uiErr != nil {
			uiErrors = append(uiErrors, *uiErr)
		}
		uiTraces[i] = uiTrace
	}

	structuredRes := structuredResponse{
		Data:   uiTraces,
		Errors: uiErrors,
	}
	aH.writeJSON(w, r, &structuredRes)
}

// getTraceFilter implements the REST API /traces/{trace-id}
// It parses trace ID from the path, fetches the trace from QueryService,
// formats it in the UI JSON format, and responds to the client.
func (aH *APIHandler) getTraceFilter(w http.ResponseWriter, r *http.Request) {
	traceID, ok := aH.parseTraceID(w, r)
	if !ok {
		return
	}
	trace, err := aH.queryService.GetTrace(r.Context(), traceID)
	if err == spanstore.ErrTraceNotFound {
		aH.handleError(w, err, http.StatusNotFound)
		return
	}
	if aH.handleError(w, err, http.StatusInternalServerError) {
		return
	}

	//for multitenancy
	if os.Getenv(envTenantCode) != "" && os.Getenv(envTenantTag) != "" {
		searchTag := os.Getenv(envTenantTag)
		tenantCode := r.Header.Get(os.Getenv(envTenantCode))
		adminTenantCode := os.Getenv(envAdminTenantCode)

		if tenantCode != "" && tenantCode != adminTenantCode {
			aH.logger.Info("search by tenant", zap.String("tenant", tenantCode))
			found := false
			for _, v := range trace.GetSpans() {
				if found {
					break
				}
				for _, tagKeyVal := range v.GetTags() {
					if searchTag == tagKeyVal.GetKey() && tenantCode == tagKeyVal.GetVStr() {
						found = true
						break
					}
				}
			}
			if !found {
				aH.handleError(w, err, http.StatusNotFound)
				return
			}
		}
	}

	var uiErrors []structuredError
	uiTrace, uiErr := aH.convertModelToUI(trace, shouldAdjust(r))
	if uiErr != nil {
		uiErrors = append(uiErrors, *uiErr)
	}

	structuredRes := structuredResponse{
		Data: []*ui.Trace{
			uiTrace,
		},
		Errors: uiErrors,
	}
	aH.writeJSON(w, r, &structuredRes)
}
