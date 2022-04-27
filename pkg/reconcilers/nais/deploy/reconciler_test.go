package nais_deploy_reconciler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nais/console/pkg/auditlogger"
	"github.com/nais/console/pkg/dbmodels"
	"github.com/nais/console/pkg/reconcilers"
	nais_deploy_reconciler "github.com/nais/console/pkg/reconcilers/nais/deploy"
	"github.com/stretchr/testify/assert"
)

func TestNaisDeployReconciler_Reconcile(t *testing.T) {
	const teamName = "myteam"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestData := &nais_deploy_reconciler.ProvisionApiKeyRequest{}
		err := json.NewDecoder(r.Body).Decode(requestData)
		assert.NoError(t, err)

		// signed with hmac signature based on timestamp and request data
		assert.Len(t, r.Header.Get("x-nais-signature"), 64)
		assert.Equal(t, teamName, requestData.Team)
		assert.Equal(t, false, requestData.Rotate)

		w.WriteHeader(http.StatusCreated)
	}))

	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	key := make([]byte, 32)
	ch := make(chan *dbmodels.AuditLog, 100)
	logger := auditlogger.New(ch)
	defer close(ch)

	reconciler := nais_deploy_reconciler.New(logger, http.DefaultClient, srv.URL, key)

	err := reconciler.Reconcile(ctx, reconcilers.Input{
		System:          nil,
		Synchronization: nil,
		Team: &dbmodels.Team{
			Slug: strp(teamName),
		},
	})

	assert.NoError(t, err)
}

func strp(s string) *string {
	return &s
}