package uritemplates_test

import (
	"github.com/Carbonfrost/joe-cli-http/uritemplates"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("Var", func() {

	Describe("Set", func() {
		DescribeTable("examples",
			func(args []string, expected *uritemplates.Var) {
				actual := new(uritemplates.Var)
				for _, a := range args {
					err := actual.Set(a)
					Expect(err).NotTo(HaveOccurred())
				}

				Expect(actual).To(PointTo(MatchFields(IgnoreExtras, Fields{
					"Name":  Equal(expected.Name),
					"Value": Equal(expected.Value),
				})))
			},
			Entry(
				"atoms",
				[]string{"array", "s", "v"},
				uritemplates.ArrayVar("s", "v"),
			),
			Entry(
				"inline",
				[]string{"array,s=v"},
				uritemplates.ArrayVar("s", "v"),
			),
			Entry(
				"inline short",
				[]string{"a,s=v"},
				uritemplates.ArrayVar("s", "v"),
			),
			Entry(
				"type and kvp",
				[]string{"a", "s=v"},
				uritemplates.ArrayVar("s", "v"),
			),
			Entry(
				"inline map",
				[]string{"map,s=v=a"},
				uritemplates.MapVar("s", map[string]any{"v": "a"}),
			),
			Entry(
				"no type",
				[]string{"v=a"},
				uritemplates.StringVar("v", "a"),
			),
		)
	})
})
