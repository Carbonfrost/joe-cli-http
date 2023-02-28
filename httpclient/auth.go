package httpclient

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/provider"
)

// AuthMode enumerates common authenticators
type AuthMode int

type Authenticator interface {
	RequiresUserInfo() bool
	Authenticate(r *http.Request, u *UserInfo) error
}

type promptForCredentials struct {
	auth Authenticator
}

// Built-in authentication modes
const (
	NoAuth AuthMode = iota
	BasicAuth
	maxAuthValue
)

var (
	authStrings = [...]string{
		"none",
		"basic",
	}
	authMarshalStrings = [...]string{
		"NO_AUTH",
		"BASIC",
	}

	Authenticators = &provider.Registry{
		Name: "authenticators",
		Providers: provider.Map{
			"basic": {},
			"none":  {},
		},
	}
)

func NewAuthenticator(name string, opts map[string]string) (Authenticator, error) {
	if name == "" {
		return NoAuth, nil
	}
	for i, a := range authStrings {
		if name == a {
			return AuthMode(i), nil
		}
	}
	for i, a := range authMarshalStrings {
		if name == a {
			return AuthMode(i), nil
		}
	}
	return nil, fmt.Errorf("unknown authenticator: %q", name)
}

func WithPromptForCredentials(auth Authenticator) Authenticator {
	return &promptForCredentials{auth}
}

func (m AuthMode) RequiresUserInfo() bool {
	return m == BasicAuth
}

func (m AuthMode) Authenticate(r *http.Request, ui *UserInfo) error {
	switch m {
	case BasicAuth:
		r.SetBasicAuth(ui.User, ui.Password)
		return nil
	case NoAuth:
		return nil
	default:
		return fmt.Errorf("unexpected auth mode %d", m)
	}
}

func (m AuthMode) String() string {
	if m >= 0 && m < maxAuthValue {
		return authStrings[int(m)]
	}
	return ""
}

// MarshalText provides the textual representation
func (m AuthMode) MarshalText() ([]byte, error) {
	switch {
	case m >= 0 && m < maxAuthValue:
		return []byte(authMarshalStrings[int(m)]), nil
	default:
		return []byte(strconv.Itoa(int(m))), nil
	}
}

// UnmarshalText converts the textual representation
func (m *AuthMode) UnmarshalText(b []byte) error {
	res, err := authModeFromName(authMarshalStrings, string(b))
	if err != nil {
		return err
	}
	*m = res
	return nil
}

func (p *promptForCredentials) Authenticate(r *http.Request, ui *UserInfo) error {
	c := r.Context().(*cli.Context)

	if ui == nil {
		ui = c.Value("user").(*UserInfo)
	}
	if p.auth.RequiresUserInfo() {
		if ui == nil {
			ui = &UserInfo{}
		}
		var err error
		if ui.User == "" {
			ui.User, err = c.ReadString("Username: ")
			if err != nil {
				return err
			}
		}
		if !ui.HasPassword {
			ui.Password, err = c.ReadPasswordString("Password: ")
			if err != nil {
				return err
			}
		}
	}
	return p.auth.Authenticate(r, ui)
}

func (*promptForCredentials) RequiresUserInfo() bool {
	return true
}

// ListAuthenticators provides an action which will list the providers
func ListAuthenticators() cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			HelpText: "List available authentication mechanisms",
			Category: requestOptions,
			Setup:    dualSetup(provider.ListProviders("authenticators")),
		},
		tagged,
	)
}

func SetAuth(v ...*provider.Value) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:      "auth",
			UsageText: "<provider>[,options...]",
			HelpText:  "Sets the authorization provider for the endpoint",
			Options:   cli.ImpliedAction,
			Category:  requestOptions,
		},
		withBinding((*Client).setAuthenticatorHelper, v),
		cli.Accessory("-", taggedProviderArgumentFlag),
		tagged,
	)
}

func taggedProviderArgumentFlag(v *provider.Value) cli.Prototype {
	proto := v.ArgumentFlag()
	proto.Setup.Uses = cli.Pipeline(
		proto.Setup.Uses,
		tagged,
	)
	return proto
}

// PromptForCredentials will display prompts for user and/or password credentials if
// authentication is required.
func PromptForCredentials() cli.Action {
	return cli.Before(cli.ActionFunc(promptForPassword))
}

func promptForPassword(c *cli.Context) error {
	client := FromContext(c)
	client.UseAuthMiddleware(WithPromptForCredentials)
	return nil
}

func SetUser(s ...*UserInfo) cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "user",
			HelpText: "Set the user and password",
			Category: requestOptions,
			Aliases:  []string{"u"},
			Setup: cli.Setup{
				Uses: cli.Implies("auth", BasicAuth.String()),
			},
		},
		withBinding((*Client).SetUser, s),
		tagged,
	)
}

func SetBasicAuth() cli.Action {
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "basic",
			HelpText: "Use Basic auth",
			Value:    new(bool),
			Category: requestOptions,
			Setup:    dualSetup(cli.BindContext(FromContext, (*Client).setAuthModeHelper, BasicAuth)),
		},
		tagged,
	)
}

func (*AuthMode) Synopsis() string {
	return "<mode>"
}

func (m *AuthMode) Set(arg string) error {
	i, err := authModeFromName(authStrings, arg)
	if err != nil {
		return err
	}
	*m = AuthMode(i)
	return nil
}

func authModeFromName(items [2]string, s string) (AuthMode, error) {
	name := strings.TrimSpace(s)
	for i, a := range items {
		if name == a {
			return AuthMode(i), nil
		}
	}
	return AuthMode(0), fmt.Errorf("unknown auth mode %q", s)
}

var _ Authenticator = AuthMode(0)
