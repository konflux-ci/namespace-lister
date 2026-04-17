package suite

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cucumber/godog"

	tcontext "github.com/konflux-ci/namespace-lister/acceptance/pkg/context"
)

func InjectHealthSteps(ctx *godog.ScenarioContext) {
	ctx.Then(`^the healthz endpoint returns 200$`, healthzReturns200)
	ctx.Then(`^the readyz endpoint returns 200$`, readyzReturns200)
}

func healthzReturns200(ctx context.Context) (context.Context, error) {
	return checkEndpoint(ctx, "/healthz")
}

func readyzReturns200(ctx context.Context) (context.Context, error) {
	return checkEndpoint(ctx, "/readyz")
}

func checkEndpoint(ctx context.Context, path string) (context.Context, error) {
	address := tcontext.NamespaceListerAddress(ctx)
	if address == "" {
		return ctx, fmt.Errorf("namespace-lister address not set in context")
	}

	tlsConfig := tcontext.TLSConfig(ctx)
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address+path, nil)
	if err != nil {
		return ctx, fmt.Errorf("error building request for %s: %w", path, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return ctx, fmt.Errorf("error requesting %s: %w", path, err)
	}
	defer resp.Body.Close()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return ctx, fmt.Errorf("error reading response body from %s: %w", path, err)
	}

	if resp.StatusCode != http.StatusOK {
		return ctx, fmt.Errorf("expected 200 from %s, got %d", path, resp.StatusCode)
	}

	return ctx, nil
}
