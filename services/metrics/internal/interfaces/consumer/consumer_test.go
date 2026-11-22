package consumer_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/interfaces/consumer"
)

// Os exemplos de contracts/ são a fonte da verdade do formato.
func TestDecodeContractExamples(t *testing.T) {
	body, err := os.ReadFile("../../../../../contracts/events/importacao/examples/treino-importado.v1.json")
	require.NoError(t, err)
	e, err := consumer.Decode(body, "TreinoImportado")
	require.NoError(t, err)
	assert.Equal(t, "corrida", e.Type)
	assert.Equal(t, 3600, e.DurationS)
	assert.InDelta(t, 10250.5, *e.DistanceM, 1e-9)

	body, err = os.ReadFile("../../../../../contracts/events/atividades/examples/atividade-corrigida.v1.json")
	require.NoError(t, err)
	_, err = consumer.Decode(body, "AtividadeCorrigida")
	require.NoError(t, err)
}

func TestDecodeRejectsWrongTypeAndGarbage(t *testing.T) {
	body, _ := os.ReadFile("../../../../../contracts/events/importacao/examples/treino-importado.v1.json")
	_, err := consumer.Decode(body, "AtividadeCorrigida")
	assert.ErrorIs(t, err, consumer.ErrInvalidMessage)
	_, err = consumer.Decode([]byte("{"), "TreinoImportado")
	assert.ErrorIs(t, err, consumer.ErrInvalidMessage)
}
