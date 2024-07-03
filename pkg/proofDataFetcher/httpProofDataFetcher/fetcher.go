package httpProofDataFetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Layr-Labs/eigenlayer-rewards-proofs/pkg/distribution"
	"github.com/Layr-Labs/eigenlayer-rewards-proofs/pkg/proofDataFetcher"
	"github.com/Layr-Labs/eigenlayer-rewards-proofs/pkg/utils"
	"io"
	"net/http"
	"strings"
)

type HttpProofDataFetcher struct {
	Client      proofDataFetcher.HTTPClient
	BaseUrl     string
	Environment string
	Network     string
}

func NewHttpProofDataFetcher(
	baseUrl string,
	environment string,
	network string,
	c proofDataFetcher.HTTPClient,
) *HttpProofDataFetcher {
	return &HttpProofDataFetcher{
		Client:      c,
		BaseUrl:     baseUrl,
		Environment: environment,
		Network:     network,
	}
}

func (h *HttpProofDataFetcher) FetchClaimAmountsForDate(ctx context.Context, date string) (*proofDataFetcher.RewardProofData, error) {

	fullUrl := h.buildClaimAmountsUrl(date)

	rawBody, err := h.handleRequest(ctx, fullUrl)
	if err != nil {
		return nil, err
	}

	return h.ProcessClaimAmountsFromRawBody(ctx, rawBody)
}

func (h *HttpProofDataFetcher) ProcessClaimAmountsFromRawBody(ctx context.Context, rawBody []byte) (*proofDataFetcher.RewardProofData, error) {
	strLines := strings.Split(string(rawBody), "\n")
	distro := distribution.NewDistribution()
	lines := []*distribution.EarnerLine{}
	for _, line := range strLines {
		if line == "" {
			continue
		}
		earner := &distribution.EarnerLine{}
		if err := json.Unmarshal([]byte(line), earner); err != nil {
			return nil, fmt.Errorf("failed to unmarshal line: %s - %w", line, err)
		}
		lines = append(lines, earner)
	}

	if err := distro.LoadLines(lines); err != nil {
		return nil, fmt.Errorf("failed to load lines: %w", err)
	}

	accountTree, tokenTree, err := distro.Merklize()
	if err != nil {
		return nil, err
	}

	proof := &proofDataFetcher.RewardProofData{
		Distribution: distro,
		AccountTree:  accountTree,
		TokenTree:    tokenTree,
		Hash:         utils.ConvertBytesToString(accountTree.Root()),
	}

	return proof, nil
}

func (h *HttpProofDataFetcher) FetchRecentSnapshotList(ctx context.Context) ([]*proofDataFetcher.Snapshot, error) {
	fullUrl := h.buildRecentSnapshotsUrl()

	rawBody, err := h.handleRequest(ctx, fullUrl)
	if err != nil {
		return nil, err
	}

	snapshots := make([]*proofDataFetcher.Snapshot, 0)
	if err := json.Unmarshal(rawBody, &snapshots); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshots: %w", err)
	}
	return snapshots, nil
}

func (h *HttpProofDataFetcher) FetchLatestSnapshot(ctx context.Context) (*proofDataFetcher.Snapshot, error) {
	snapshots, err := h.FetchRecentSnapshotList(ctx)

	if err != nil {
		return nil, err
	}
	if len(snapshots) == 0 {
		return nil, fmt.Errorf("no snapshots found")
	}
	return snapshots[0], nil
}

func (h *HttpProofDataFetcher) FetchPostedRewards(ctx context.Context) ([]*proofDataFetcher.SubmittedRewardRoot, error) {
	fullUrl := h.buildPostedRewardsUrl()

	rawBody, err := h.handleRequest(ctx, fullUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch posted rewards: %w", err)
	}

	rewards := make([]*proofDataFetcher.SubmittedRewardRoot, 0)
	if err := json.Unmarshal(rawBody, &rewards); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rewards: %w", err)
	}
	return rewards, nil
}

func (h *HttpProofDataFetcher) handleRequest(ctx context.Context, fullUrl string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to form request: %w", err)
	}

	res, err := h.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed %w", err)
	}
	defer res.Body.Close()

	rawBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("Received error code '%d'", res.StatusCode)
	}

	return rawBody, nil
}

func (h *HttpProofDataFetcher) buildRecentSnapshotsUrl() string {
	// <baseurl>/<env>/<network>/recent-snapshots.json
	return fmt.Sprintf("%s/%s/%s/recent-snapshots.json",
		h.BaseUrl,
		h.Environment,
		h.Network,
	)
}

func (h *HttpProofDataFetcher) buildClaimAmountsUrl(snapshotDate string) string {
	// <baseurl>/<env>/<network>/<snapshot_date>/claim-amounts.json
	return fmt.Sprintf("%s/%s/%s/%s/claim-amounts.json",
		h.BaseUrl,
		h.Environment,
		h.Network,
		snapshotDate,
	)
}

func (h *HttpProofDataFetcher) buildPostedRewardsUrl() string {
	// <baseurl>/<env>/<network>/submitted-payments.json
	return fmt.Sprintf("%s/%s/%s/submitted-payments.json",
		h.BaseUrl,
		h.Environment,
		h.Network,
	)
}
