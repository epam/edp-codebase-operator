package codebaseimagestream

import (
	"errors"
	"time"

	"github.com/go-logr/logr"

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
