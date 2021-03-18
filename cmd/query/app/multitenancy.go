package app

import (
	"net/http"
	"os"
	"strings"

	"github.com/jaegertracing/jaeger/model"
	ui "github.com/jaegertracing/jaeger/model/json"
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
		tenantCode := strings.ToLower(r.Header.Get(os.Getenv(envTenantCode)))
		adminTenantCode := strings.ToLower(r.Header.Get(os.Getenv(envAdminTenantCode)))

		if tenantCode != "" && tenantCode != adminTenantCode {
			tQuery.Tags[os.Getenv(envTenantTag)] = tenantCode
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
