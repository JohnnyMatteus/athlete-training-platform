package consumer_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/interfaces/consumer"
)

func TestDecodeLinkContractExamples(t *testing.T) {
	_, l, err := consumer.DecodeLink(readExample(t, "../../../../../contracts/events/atleta/examples/vinculo-consentido.v1.json"))
	require.NoError(t, err)
	assert.True(t, l.Active)
	assert.Equal(t, []string{"treinos", "plano"}, l.Categories)

	_, l, err = consumer.DecodeLink(readExample(t, "../../../../../contracts/events/atleta/examples/vinculo-revogado.v1.json"))
	require.NoError(t, err)
	assert.False(t, l.Active)
}

func TestDecodeLinkRejectsUnknownType(t *testing.T) {
	_, _, err := consumer.DecodeLink([]byte(`{"tipo":"Outro"}`))
	assert.ErrorIs(t, err, consumer.ErrInvalidMessage)
}

func readExample(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return b
}
