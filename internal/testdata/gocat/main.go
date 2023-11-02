// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"io"
	"log"
	"os"
)

func main() {
	flag.Parse()

	filename := flag.Arg(0)
	if filename == "" {
		log.Fatalln("Must specify filename")
	}

	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Error opening file %q: %v\n", filename, err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(os.Stdout, f); err != nil {
		log.Fatalf("Error copying file content: %v\n", err)
	}
}
