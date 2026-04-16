// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uritemplates

import (
	"context"
	"fmt"
	"io"
	"reflect"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/value"
)

// Expander provides URI template expansion functionality that can be initialized
// and configured within a CLI app. The expander is used within the Uses pipeline
// where it registers itself as a context service.
type Expander struct {
	cli.Action

	vars     Vars
	partial  bool
	template *URITemplate
}

// Option is an option to configure the expander
// Option can be used as an Action, typically within the Uses or Before pipeline.
type Option interface {
	cli.Action
	apply(*Expander)
}

type option[T any] struct {
	val T
	fn  func(*Expander, T) error
}

func (o option[_]) Execute(ctx context.Context) error {
	o.apply(FromContext(ctx))
	return nil
}

func (o option[_]) apply(e *Expander) {
	o.fn(e, o.val)
}

type contextKey string

const expanderContextKey contextKey = "uritemplates_expander"

var (
	tagged  = cli.Data(SourceAnnotation())
	pkgPath = reflect.TypeFor[Var]().PkgPath()

	impliedOptions = []Option{
		WithDefaultAction(),
	}
)

// SourceAnnotation gets the name and value of the annotation added to the Data
// of all flags that are initialized from this package
func SourceAnnotation() (string, string) {
	return "Source", pkgPath
}

// New creates a new URI template expander with the given options.
func New(options ...Option) *Expander {
	e := &Expander{
		vars: Vars{},
	}
	e.Apply(impliedOptions...)
	e.Apply(options...)
	return e
}

func (e *Expander) Pipeline() cli.Action {
	return e.Action
}

// WithAction sets up the action to use when Expander is added to a pipeline
func WithAction(a cli.Action) Option {
	return withAdapter((*Expander).setAction, a)
}

// WithDefaultAction sets up the default action to use when Expander.
func WithDefaultAction() Option {
	return option[cli.Action]{
		nil, func(e *Expander, _ cli.Action) error {
			e.Action = cli.Pipeline(
				ContextValue(e),
				FlagsAndArgs(),
			)
			return nil
		},
	}
}

// Apply applies the given options to the expander
func (e *Expander) Apply(opts ...Option) {
	for _, o := range opts {
		o.apply(e)
	}
}

// FromContext obtains the expander from the context.
func FromContext(ctx context.Context) *Expander {
	return ctx.Value(expanderContextKey).(*Expander)
}

// ContextValue returns an action that adds the expander to the context
func ContextValue(e *Expander) cli.Action {
	return cli.WithContextValue(expanderContextKey, e)
}

// WithURITemplateVar adds a template variable
func WithURITemplateVar(v *Var) Option {
	return withAdapter((*Expander).AddURITemplateVar, v)
}

// WithURITemplateVars adds multiple template variables
func WithURITemplateVars(v *Vars) Option {
	return withAdapter((*Expander).UpdateURITemplateVars, v)
}

// WithPartialExpand sets whether to partially expand the template
func WithPartialExpand(b bool) Option {
	return withAdapter((*Expander).SetPartialExpand, b)
}

// AddURITemplateVar adds a template variable to the expander
func (e *Expander) AddURITemplateVar(v *Var) error {
	e.vars.Add(v)
	return nil
}

// UpdateURITemplateVars updates the template variables in the expander
func (e *Expander) UpdateURITemplateVars(v *Vars) error {
	return e.vars.Update(*v)
}

// SetPartialExpand sets whether to partially expand the template
func (e *Expander) SetPartialExpand(b bool) error {
	e.partial = b
	return nil
}

// SetTemplate sets the template to expand
func (e *Expander) SetTemplate(v *URITemplate) error {
	e.template = v
	return nil
}

func (e *Expander) setAction(v cli.Action) error {
	e.Action = v
	return nil
}

// Template returns the template to expand
func (e *Expander) Template() *URITemplate {
	return e.template
}

// Vars returns the template variables
func (e *Expander) Vars() Vars {
	return e.vars
}

// Partial returns whether partial expansion is enabled
func (e *Expander) Partial() bool {
	return e.partial
}

// Expand performs the expansion
func (e *Expander) Expand() (string, error) {
	tpl := e.Template()

	if e.Partial() {
		rr, err := tpl.PartialExpand(e.Vars())
		return fmt.Sprint(rr), err
	}

	return tpl.Expand(e.Vars())
}

func (e *Expander) expandAndPrint(stdout io.Writer) error {
	rr, err := e.Expand()
	fmt.Fprintln(stdout, rr)
	return err
}

// ExpandAndPrint returns an action that expands the URI template and
// prints the output
func ExpandAndPrint() cli.Action {
	return cli.Pipeline(
		cli.Prototype{
			Uses: New(),
		},
		bind.Call2((*Expander).expandAndPrint, bind.FromContext(FromContext), bind.Stdout()),
	)
}

// FlagsAndArgs returns an action that sets up flags and arguments for URI template expansion
func FlagsAndArgs() cli.Action {
	return cli.Pipeline(
		cli.AddFlags([]*cli.Flag{
			{Uses: SetURITemplateVar()},
			{Uses: SetURITemplateVars()},
			{Uses: SetPartialExpand()},
		}...),

		cli.AddArg(nil, SetTemplate()),
	)
}

// SetTemplate sets the template used
func SetTemplate(v ...*URITemplate) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:  "template",
			NArg:  cli.TakeUntilNextFlag,
			Value: new(URITemplate),
		},
		bind.Call2((*Expander).SetTemplate, bind.FromContext(FromContext), bind.Exact(v...)),
	)
}

// SetURITemplateVar returns an action that sets up a flag for specifying template variables
func SetURITemplateVar(v ...*Var) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "param",
			Aliases:  []string{"T"},
			HelpText: "Specify a value used to fill the template",
			Value:    new(Var),
			Options:  cli.EachOccurrence,
		},
		bind.Call2((*Expander).AddURITemplateVar, bind.FromContext(FromContext), bind.Exact(v...)),
		tagged,
	)
}

// SetURITemplateVars returns an action that sets up a flag for specifying template variables from JSON
func SetURITemplateVars(v ...*Vars) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:      "params",
			Aliases:   []string{"t"},
			UsageText: "expr|@file",
			HelpText:  "Specify a template parameters using abbreviated syntax or from a JSON file",
			Value:     value.JSON(&Vars{}),
			Options:   cli.EachOccurrence | cli.AllowFileReference,
		},
		bind.Call2((*Expander).UpdateURITemplateVars, bind.FromContext(FromContext), bind.Exact(v...)),
		tagged,
	)
}

// SetPartialExpand returns an action that sets up a flag for enabling partial expansion
func SetPartialExpand(b ...bool) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "partial",
			Aliases:  []string{"P"},
			Value:    cli.Bool(),
			HelpText: "When set, partially expand the template by preserving missing variables",
		},
		bind.Call2((*Expander).SetPartialExpand, bind.FromContext(FromContext), bind.Exact(b...)),
		tagged,
	)
}

func withAdapter[T any](fn func(*Expander, T) error, value T) Option {
	return option[T]{value, fn}
}
