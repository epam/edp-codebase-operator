package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestMakeChain(t *testing.T) {
	t.Parallel()

	c := MakeChain(
		context.Background(),
		fake.NewClientBuilder().Build(),
	)

	assert.NotNil(t, c)
}

func TestMakeDeletionChain(t *testing.T) {
	t.Parallel()

	c := MakeDeletionChain(
		context.Background(),
		fake.NewClientBuilder().Build(),
	)

	assert.NotNil(t, c)
}
