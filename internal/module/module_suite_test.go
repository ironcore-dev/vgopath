// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package module_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var gocatExecutable string

var _ = BeforeSuite(func() {
	gocatDir, err := gexec.Build(".", "-C", filepath.Join("..", "testdata", "gocat"))
	Expect(err).NotTo(HaveOccurred())
	gocatExecutable = filepath.Join(gocatDir, "gocat")
	DeferCleanup(gexec.CleanupBuildArtifacts)
})

func TestInternal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Internal Suite")
}
