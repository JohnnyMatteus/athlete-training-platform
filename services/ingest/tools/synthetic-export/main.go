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

	sum, err := generate(*out, *seed, *sizeMB)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(sum)
}

// generate grava o export em path e devolve o SHA-256. O erro de Close importa: é o último
// ponto em que uma falha de escrita no disco aparece.
func generate(path string, seed, sizeMB int64) (sum string, err error) {
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			sum, err = "", cerr
		}
	}()
	h := sha256.New()
	if err := fixtures.WriteAppleExport(io.MultiWriter(f, h), fixtures.AppleOptions{Seed: seed, TargetXMLBytes: sizeMB << 20}); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
