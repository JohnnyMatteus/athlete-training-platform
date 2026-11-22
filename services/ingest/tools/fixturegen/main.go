// Comando fixturegen grava os arquivos de teste de contrato em testdata/.
// Uso: go run ./tools/fixturegen [diretório]  (padrão: testdata)
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/testkit/fixtures"
)

func main() {
	root := "testdata"
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	files := map[string][]byte{
		"fit/bike-potencia.fit":     fixtures.FitBikePower(),
		"fit/natacao-piscina.fit":   fixtures.FitSwimPool(),
		"fit/outro-fabricante.fit":  fixtures.FitOtherVendor(),
		"apple/export-reduzido.zip": fixtures.AppleReduced(),
		"gpx/corrida.gpx":           []byte(fixtures.GPXRun("running", fixtures.TimeUTC)),
		"tcx/bike.tcx":              []byte(fixtures.TCXBike(fixtures.TimeUTC)),
	}
	for name, data := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			fail(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			fail(err)
		}
		fmt.Println("gerado", path)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
