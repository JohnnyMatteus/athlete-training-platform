package consumer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/interfaces/consumer"
)

// mesmo corpo do exemplo em contracts/events/importacao/examples
const example = `{"event_id":"0b9a8c7d-6e5f-4a3b-9c2d-1e0f9a8b7c6d","tipo":"ImportacaoRecebida","versao":1,
"ocorrido_em":"2026-03-02T10:15:30Z","import_id":"3c1d2e4f-0b1a-4c2d-8e3f-9a8b7c6d5e40",
"dados":{"atleta_id":"2f0f0a2e-6a53-4c3a-9d8e-1b7e6c5d4a01","tipo_arquivo":"zip","bucket":"uploads",
"object_key":"a1b2c3","tamanho_bytes":1024,"sha256":"0000000000000000000000000000000000000000000000000000000000000000"}}`

func TestDecode(t *testing.T) {
	m, err := consumer.Decode([]byte(example), 2)
	require.NoError(t, err)
	assert.Equal(t, "zip", m.FileType)
	assert.Equal(t, "uploads", m.Bucket)
	assert.Equal(t, "a1b2c3", m.ObjectKey)
	assert.Equal(t, 3, m.Attempt, "x-delivery-count 2 é a 3ª entrega")
}

func TestDecodeInvalid(t *testing.T) {
	for _, body := range []string{`nope`, `{"tipo":"Outro","versao":1}`, `{"tipo":"ImportacaoRecebida","versao":2}`} {
		_, err := consumer.Decode([]byte(body), 0)
		assert.ErrorIs(t, err, consumer.ErrInvalidMessage, body)
	}
}
