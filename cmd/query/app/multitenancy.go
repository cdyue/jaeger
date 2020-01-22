// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	tenantCode      = "JAGER_TENANT_CODE"       //tenant
	adminTenantCode = "JAGER_ADMIN_TENANT_CODE" //admin tenant, admin can visit all trace.
	tenantTag       = "JAGER_TENANT_TAG"        //tenant tag for search
)

func (aH *APIHandler) searchForMultitenancy(w http.ResponseWriter, r *http.Request) {
	tQuery, err := aH.queryParser.parse(r)
	if aH.handleError(w, err, http.StatusBadRequest) {
		return
	}
	//for multitenancy
	if os.Getenv(tenantCode) != "" && os.Getenv(tenantTag) != "" {
		tenantCode := strings.ToLower(r.Header.Get(os.Getenv(tenantCode)))
		if tenantCode != "" && tenantCode != os.Getenv(adminTenantCode) {
			tQuery.Tags[os.Getenv(tenantTag)] = tenantCode
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
