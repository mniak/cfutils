package codefresh

import (
	"context"

	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
)

func (c *Client) BuildProgress(ctx context.Context, buildID string) (*Progress, error) {
	api := c.Rest()

	build, err := api.Workflow().Get(buildID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get build %s", buildID)
	}
	ref, err := api.Progress().Get(build.Progress)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get progress %s", build.Progress)
	}

	var progress Progress
	resp, err := resty.New().R().
		SetContext(ctx).
		SetResult(&progress).
		Get(ref.Location.URL)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to download the progress document")
	}
	if resp.IsError() {
		return nil, errors.Errorf("failed to download the progress document: %s", resp.Status())
	}
	return &progress, nil
}