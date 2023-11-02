// Copyright 2022 IronCore authors
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

package link_test

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"

	. "github.com/ironcore-dev/vgopath/internal/link"
	"github.com/ironcore-dev/vgopath/internal/module"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Internal", func() {
	var (
		tmpDir                                                            string
		moduleA, moduleB, moduleB1, moduleB11, moduleB2, moduleC, moduleD module.Module
	)
	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "test")
		Expect(err).NotTo(HaveOccurred())

		moduleA = module.Module{
			Path: "a",
			Dir:  "/tmp/a",
			Main: true,
		}
		moduleB = module.Module{
			Path: "example.org/b",
			Dir:  "/tmp/example.org/b",
		}
		moduleB1 = module.Module{
			Path: "example.org/b/1",
			Dir:  "/tmp/example.org/b/1",
		}
		moduleB11 = module.Module{
			Path: "example.org/b/1/1",
			Dir:  "/tmp/example.org/b/1/1",
		}
		moduleB2 = module.Module{
			Path: "example.org/b/2",
			Dir:  "/tmp/example.org/b/2",
		}
		moduleC = module.Module{
			Path: "example.org/user/c",
			Dir:  "/tmp/example.org/user/c",
		}
		moduleD = module.Module{
			Path: "example.org/d",
		}
	})
	AfterEach(func() {
		if tmpDir != "" {
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		}
	})

	Describe("BuildModuleNodes", func() {
		It("should correctly build the nodes", func() {
			nodes, err := BuildModuleNodes([]module.Module{moduleA, moduleB, moduleB1, moduleB11, moduleB2, moduleC})
			Expect(err).NotTo(HaveOccurred())
			Expect(nodes).To(ConsistOf(
				Node{
					Segment: "a",
					Module:  &moduleA,
				},
				Node{
					Segment: "example.org",
					Children: []Node{
						{
							Segment: "b",
							Module:  &moduleB,
							Children: []Node{
								{
									Segment: "1",
									Module:  &moduleB1,
									Children: []Node{
										{
											Segment: "1",
											Module:  &moduleB11,
										},
									},
								},
								{
									Segment: "2",
									Module:  &moduleB2,
								},
							},
						},
						{
							Segment: "user",
							Children: []Node{
								{
									Segment: "c",
									Module:  &moduleC,
								},
							},
						},
					},
				},
			))
		})

		It("should error on invalid module paths", func() {
			_, err := BuildModuleNodes([]module.Module{{Path: ""}})
			Expect(err).To(HaveOccurred())
		})

		It("should error if there are modules pointing to the same path", func() {
			_, err := BuildModuleNodes([]module.Module{{Path: "foo"}, {Path: "foo"}})
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("FilterModulesWithoutDir", func() {
		It("should correctly filter the modules", func() {
			mods := FilterModulesWithoutDir([]module.Module{moduleA, moduleB, moduleD})
			Expect(mods).To(Equal([]module.Module{moduleA, moduleB}))
		})
	})

	Context("Link", func() {
		var (
			srcGopathDir string
			dstGopathDir string
		)
		BeforeEach(func() {
			srcGopathDir = filepath.Join(tmpDir, "srcGopath")
			dstGopathDir = filepath.Join(tmpDir, "dstGopath")

			Expect(os.MkdirAll(srcGopathDir, 0777)).To(Succeed())
			Expect(os.MkdirAll(dstGopathDir, 0777)).To(Succeed())
		})

		Describe("GoBin", func() {
			var (
				srcGoBinDir string
				dstGoBinDir string
			)
			BeforeEach(func() {
				srcGoBinDir = filepath.Join(srcGopathDir, "bin")
				Expect(os.MkdirAll(srcGoBinDir, 0777)).To(Succeed())

				dstGoBinDir = filepath.Join(dstGopathDir, "bin")
				Expect(os.MkdirAll(dstGopathDir, 0777)).To(Succeed())
			})

			It("should correctly link go bin", func() {
				defer setEnvAndRevert("GOBIN", "")()
				defer setAndRevert(&build.Default.GOPATH, srcGopathDir)()

				Expect(GoBin(dstGopathDir)).To(Succeed())
				Expect(dstGoBinDir).To(BeASymlinkTo(srcGoBinDir))
			})

			It("should correctly link go bin if GOBIN is set", func() {
				defer setEnvAndRevert("GOBIN", srcGoBinDir)()

				Expect(GoBin(dstGopathDir)).To(Succeed())
				Expect(dstGoBinDir).To(BeASymlinkTo(srcGoBinDir))
			})
		})

		Describe("GoPkg", func() {
			var (
				srcGoPkgDir string
				dstGoPkgDir string
			)
			BeforeEach(func() {
				srcGoPkgDir = filepath.Join(srcGopathDir, "pkg")
				Expect(os.MkdirAll(srcGoPkgDir, 0777)).To(Succeed())

				dstGoPkgDir = filepath.Join(dstGopathDir, "pkg")
				Expect(os.MkdirAll(dstGopathDir, 0777)).To(Succeed())
			})

			It("should correctly link go pkg", func() {
				defer setAndRevert(&build.Default.GOPATH, srcGopathDir)()

				Expect(GoPkg(dstGopathDir)).To(Succeed())
				Expect(dstGoPkgDir).To(BeASymlinkTo(srcGoPkgDir))
			})
		})
	})
})

func BeASymlinkTo(filename string) types.GomegaMatcher {
	return &beASymlinkToMatcher{filename}
}

type beASymlinkToMatcher struct {
	filename string
}

func (m *beASymlinkToMatcher) Match(actual interface{}) (success bool, err error) {
	actualFilename, ok := actual.(string)
	if !ok {
		return false, fmt.Errorf("IsSymlinkTo expects a filename string")
	}

	actualStat, err := os.Lstat(actualFilename)
	if err != nil {
		return false, err
	}

	if (actualStat.Mode() & os.ModeSymlink) != os.ModeSymlink {
		return false, nil
	}

	tgt, err := os.Readlink(actualFilename)
	if err != nil {
		return false, err
	}

	return tgt == m.filename, nil
}

func (m *beASymlinkToMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%v\nto be a symlink to\n\t%s", actual, m.filename)
}

func (m *beASymlinkToMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%v\nnot to be a symlink to\n\t%s", actual, m.filename)
}

func setEnvAndRevert(key, value string) func() {
	oldValue := os.Getenv(key)
	if value == "" {
		_ = os.Unsetenv(key)
	} else {
		_ = os.Setenv(key, value)
	}
	return func() {
		if oldValue == "" {
			_ = os.Unsetenv(key)
		} else {
			_ = os.Setenv(key, oldValue)
		}
	}
}

func setAndRevert(pointerToString *string, newValue string) func() {
	oldValue := *pointerToString
	*pointerToString = newValue
	return func() {
		*pointerToString = oldValue
	}
}
