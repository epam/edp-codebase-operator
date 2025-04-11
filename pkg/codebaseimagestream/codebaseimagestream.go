package codebaseimagestream

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func GetLastTag(tags []codebaseApi.Tag, log logr.Logger) (codebaseApi.Tag, error) {
	var (
		latestTag     codebaseApi.Tag
		latestTagTime = time.Time{}
	)

	for i, s := range tags {
		current, err := time.Parse(time.RFC3339, tags[i].Created)
		if err != nil {
			log.Error(err, "Failed to parse tag created time. Skip tag.", "tag", s.Name)
		}

		if current.After(latestTagTime) {
			latestTagTime = current
			latestTag = s
		}
	}

	if latestTag.Name == "" {
		return latestTag, errors.New("latest tag is not found")
	}

	return latestTag, nil
}

var ErrCodebaseImageStreamNotFound = errors.New("CodebaseImageStream not found")

func GetCodebaseImageStreamByCodebaseBaseBranchName(
	ctx context.Context,
	k8sCl client.Client,
	codebaseBranchName string,
	namespace string,
) (*codebaseApi.CodebaseImageStream, error) {
	var codebaseImageStreamList codebaseApi.CodebaseImageStreamList

	if err := k8sCl.List(
		ctx,
		&codebaseImageStreamList,
		client.InNamespace(namespace),
		client.MatchingLabels{
			codebaseApi.CodebaseImageStreamCodebaseBranchLabel: codebaseBranchName,
		},
	); err != nil {
		return nil, fmt.Errorf("failed to get CodebaseImageStream by label: %w", err)
	}

	if len(codebaseImageStreamList.Items) == 0 {
		return nil, fmt.Errorf(
			"failed to get CodebaseImageStream for CodebaseBranch %s: %w",
			codebaseBranchName,
			ErrCodebaseImageStreamNotFound,
		)
	}

	if len(codebaseImageStreamList.Items) > 1 {
		return nil, fmt.Errorf("multiple CodebaseImageStream found for CodebaseBranch %s", codebaseBranchName)
	}

	return &codebaseImageStreamList.Items[0], nil
}
