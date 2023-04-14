// Copyright 2022 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package module_test

import (
	"io"
	"os/exec"
	"path/filepath"

	"github.com/onmetal/vgopath/internal/module"
	"github.com/onmetal/vgopath/internal/testdata"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Internal", func() {
	Describe("OpenGoList", func() {
		It("should create a read closer that can be closed", FlakeAttempts(50), func() {
			rc, err := module.OpenGoList(module.InDir(filepath.Join("..", "..")))
			Expect(err).NotTo(HaveOccurred())
			Expect(rc).NotTo(BeNil())

			Expect(rc.Close()).To(Succeed())
		})

		It("should error on the first read if modules are not present", func() {
			rc, err := module.OpenGoList(module.InDir(GinkgoT().TempDir()))
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(rc.Close)

			_, err = rc.Read(make([]module.Module, 2))
			Expect(err).To(HaveOccurred())
		})

		It("should return the modules and return io.EOF when done", func() {
			By("opening the ginkgo temp dir")
			rc, err := module.OpenGoList(module.InDir(filepath.Join("..", "testdata", "gocat")))
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(rc.Close)

			modules := make([]module.Module, 2)
			n, err := rc.Read(modules)
			Expect(err).To(MatchError(io.EOF))
			Expect(n).To(Equal(1))
		})

		It("should parse the command data as JSON stream", func() {
			cmd := func() *exec.Cmd {
				return exec.Command(gocatExecutable, filepath.Join("..", "testdata", "modules.json.stream"))
			}
			rc, err := module.OpenGoList(&module.OpenGoListOptions{Command: cmd})
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(rc.Close)

			modules := make([]module.Module, 4)
			n, err := rc.Read(modules)
			Expect(err).To(MatchError(io.EOF))
			Expect(n).To(Equal(3))
			Expect(modules[:3]).To(Equal(testdata.Modules))
		})
	})
})
