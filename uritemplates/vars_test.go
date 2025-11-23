// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package uritemplates_test

import (
	"context"
	"io/fs"
	"testing/fstest"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli-http/uritemplates"
	"github.com/Carbonfrost/joe-cli/value"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Vars", func() {

	Describe("Set", func() {
		var testFileSystem = func() fs.FS {
			return fstest.MapFS{
				"vars.json": {
					Data: []byte(`{
						            "id": 420,
						            "terms": ["asdf", "jkl;"]
						        }`),
				},
			}
		}()

		It("parses vars from JSON file", func() {
			actual := &uritemplates.Vars{}
			app := &cli.App{
				FS: testFileSystem,
				Flags: []*cli.Flag{
					{
						Name:    "V",
						Value:   value.JSON(actual),
						Options: cli.AllowFileReference,
					},
				},
			}

			args, _ := cli.Split("app -V @vars.json")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(&uritemplates.Vars{
				"id":    float64(420),
				"terms": []any{"asdf", "jkl;"},
			}))
		})

		DescribeTable("examples",
			func(args []string, expected *uritemplates.Vars) {
				actual := new(uritemplates.Vars)
				for _, a := range args {
					err := actual.Set(a)
					Expect(err).NotTo(HaveOccurred())
				}

				Expect(actual).To(Equal(expected))
			},
			Entry("atoms", []string{"id=420"}, &uritemplates.Vars{"id": "420"}),
			Entry("array",
				[]string{"terms=[a,b]"},
				&uritemplates.Vars{"terms": []any{"a", "b"}},
			),
			Entry("array ws",
				[]string{"terms=[a, b, c ]"},
				&uritemplates.Vars{"terms": []any{"a", "b", "c"}},
			),
			Entry("array with ws",
				[]string{"terms=[a with ws, b, c ]"},
				&uritemplates.Vars{"terms": []any{"a with ws", "b", "c"}},
			),
			Entry("map",
				[]string{"map={s:a,t:b}"},
				&uritemplates.Vars{
					"map": map[string]any{
						"s": "a",
						"t": "b",
					}}),
			Entry("bare name", []string{"id"}, &uritemplates.Vars{"id": "id"}),
		)
	})

	Describe("String", func() {
		DescribeTable("examples",
			func(v *uritemplates.Vars, expected string) {
				Expect(v.String()).To(Equal(expected))
			},
			Entry("atoms", &uritemplates.Vars{"id": "420"}, "id=420"),
			Entry("array",
				&uritemplates.Vars{"terms": []any{"a", "b"}},
				"terms=[a,b]",
			),
			Entry("map",
				&uritemplates.Vars{
					"map": map[string]any{
						"s": "a",
						"t": "b",
					}},
				"map={s:a,t:b}",
			),
			Entry("string", &uritemplates.Vars{"id": "en"}, "id=en"),
			Entry("bare name", &uritemplates.Vars{"id": "id"}, "id"),
		)
	})
})
