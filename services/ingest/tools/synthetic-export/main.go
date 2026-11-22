// Comando synthetic-export gera um export.zip fictício do app Saúde, reprodutível por semente.
// O tamanho é o de export.xml descompactado (o que o parser lê); o .zip fica menor.
//
//	go run ./tools/synthetic-export -size-mb 1024 -seed 42 -out /tmp/export-1gb.zip
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/testkit/fixtures"
)

func main() {
	sizeMB := flag.Int64("size-mb", 100, "tamanho de export.xml em MB")
	seed := flag.Int64("seed", 1, "semente (mesma semente, mesmos bytes)")
	out := flag.String("out", "export.zip", "arquivo de saída")
	flag.Parse()

	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	h := sha256.New()
	if err := fixtures.WriteAppleExport(io.MultiWriter(f, h), fixtures.AppleOptions{Seed: *seed, TargetXMLBytes: *sizeMB << 20}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(hex.EncodeToString(h.Sum(nil)))
}
