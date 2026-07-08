package modeldiscovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	neturl "net/url"
	"strings"
	"time"
)

// ErrForbidden is returned (wrapped) by GetService when the Unity Catalog API
// responds with 403. The orchestrator uses it to skip a service rather than
// hard-fail, hedging the assumption that the LIST endpoint is filtered by the
// caller's EXECUTE grants.
var ErrForbidden = errors.New("forbidden")

// modelServicesPath is the Unity Catalog model-services collection endpoint.
const modelServicesPath = "/api/2.1/unity-catalog/model-services"

// listPageSize is the page_size requested when listing model-services. It is
// cheap insurance against a low default; pagination is still followed.
const listPageSize = 200

// defaultTimeout bounds a single discovery HTTP request. Discovery runs only at
// config/desktop/doctor time, never on the launch hot path, so a generous but
// finite ceiling is right: it prevents a hung gateway from blocking those
// commands forever (context.Background carries no deadline of its own).
const defaultTimeout = 30 * time.Second

// maxResponseBody caps how much of a model-services response body is buffered
// before JSON decoding. Model-services lists are small; the cap defends against
// a compromised or misbehaving gateway returning an unbounded body.
const maxResponseBody = 8 << 20 // 8 MiB

// NewClient returns the HTTP client discovery callers should use. It bounds each
// request with a timeout and refuses cross-host redirects so the OAuth bearer
// token can never be replayed against an attacker-chosen host (Go already
// strips the Authorization header on a cross-host redirect, but refusing the
// redirect outright also blocks the SSRF probe). Prefer this over
// http.DefaultClient, which has no timeout and follows redirects.
func NewClient() *http.Client {
	return &http.Client{
		Timeout: defaultTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 0 && req.URL.Host != via[0].URL.Host {
				return fmt.Errorf("model-services: refusing cross-host redirect to %s", req.URL.Host)
			}
			return nil
		},
	}
}

// wireDest mirrors a single routing.destinations[] entry on the wire.
type wireDest struct {
	Name string `json:"name"`
}

// wireService mirrors a model-service object on the wire, used for both LIST
// entries and the individual GET response.
type wireService struct {
	FullName          string      `json:"full_name"`
	SupportedAPITypes []string    `json:"supported_api_types"`
	Routing           wireRouting `json:"routing"`
}

// wireRouting mirrors the routing block of a model-service.
type wireRouting struct {
	Destinations []wireDest `json:"destinations"`
}

// wireListResponse mirrors the LIST response envelope.
type wireListResponse struct {
	ModelServices []wireService `json:"model_services"`
	NextPageToken string        `json:"next_page_token"`
}

// toService converts a wire representation into the exported Service, computing
// the catalog and parsing every routing destination.
func (w wireService) toService() Service {
	svc := Service{
		FQN:               w.FullName,
		Catalog:           catalogOf(w.FullName),
		SupportedAPITypes: w.SupportedAPITypes,
	}
	for _, d := range w.Routing.Destinations {
		svc.Destinations = append(svc.Destinations, parseDestination(d.Name))
	}
	return svc
}

// catalogOf returns the first dot-segment of a fully-qualified name.
func catalogOf(fqn string) string {
	if i := strings.IndexByte(fqn, '.'); i >= 0 {
		return fqn[:i]
	}
	return fqn
}

// ListServices returns every model-service visible to the caller, following
// next_page_token pagination. Non-2xx responses yield an error that includes the
// status code. An empty result is not itself an error.
func ListServices(ctx context.Context, client *http.Client, host, token string) ([]Service, error) {
	var services []Service
	pageToken := ""
	for {
		url := fmt.Sprintf("%s%s?page_size=%d", strings.TrimRight(host, "/"), modelServicesPath, listPageSize)
		if pageToken != "" {
			// pageToken is echoed from the gateway response — escape it so a
			// crafted value cannot inject extra query parameters.
			url += "&page_token=" + neturl.QueryEscape(pageToken)
		}

		body, err := doGet(ctx, client, url, token)
		if err != nil {
			return nil, err
		}

		var resp wireListResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("model-services: decode list response: %w", err)
		}
		for _, w := range resp.ModelServices {
			services = append(services, w.toService())
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return services, nil
}

// GetService fetches a single model-service by FQN. A 403 response is returned
// wrapped around ErrForbidden so callers can skip the service; other non-2xx
// responses yield an error including the status code.
func GetService(ctx context.Context, client *http.Client, host, token, fqn string) (Service, error) {
	// fqn comes verbatim from the LIST response — escape it as a single path
	// segment so a crafted service name cannot traverse to another API path.
	url := fmt.Sprintf("%s%s/%s", strings.TrimRight(host, "/"), modelServicesPath, neturl.PathEscape(fqn))
	body, err := doGet(ctx, client, url, token)
	if err != nil {
		return Service{}, err
	}

	var w wireService
	if err := json.Unmarshal(body, &w); err != nil {
		return Service{}, fmt.Errorf("model-services: decode get response for %q: %w", fqn, err)
	}
	return w.toService(), nil
}

// doGet performs an authenticated GET and returns the response body. It maps a
// 403 to ErrForbidden and any other non-2xx status to a status-bearing error.
func doGet(ctx context.Context, client *http.Client, url, token string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("model-services: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("model-services: request %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("model-services: read response body: %w", err)
	}

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("model-services: GET %s: %w", url, ErrForbidden)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("model-services: GET %s: unexpected status %d", url, resp.StatusCode)
	}
	return body, nil
}

// Discover lists model-services, enriches any LIST entry that lacks
// supported_api_types via an individual GetService call, and resolves the
// newest model per family. A GetService that returns ErrForbidden causes that
// service to be skipped with a logged note; any other GetService error is
// propagated. A ListServices failure is returned directly.
func Discover(ctx context.Context, client *http.Client, host, token string, pins Pins) (ModelSet, []Unresolved, error) {
	services, err := ListServices(ctx, client, host, token)
	if err != nil {
		return ModelSet{}, nil, err
	}

	enriched := make([]Service, 0, len(services))
	for _, svc := range services {
		if len(svc.SupportedAPITypes) == 0 {
			full, gerr := GetService(ctx, client, host, token, svc.FQN)
			if gerr != nil {
				if errors.Is(gerr, ErrForbidden) {
					log.Printf("modeldiscovery: skipping service %q: %v", svc.FQN, gerr)
					continue
				}
				return ModelSet{}, nil, gerr
			}
			enriched = append(enriched, full)
			continue
		}
		enriched = append(enriched, svc)
	}

	set, unresolved := Resolve(enriched, pins)
	return set, unresolved, nil
}
