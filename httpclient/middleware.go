package httpclient

import (
	"net/http"
)

type Middleware interface {
	Handle(req *http.Request) error
}

type MiddlewareFunc func(req *http.Request) error

func (f MiddlewareFunc) Handle(req *http.Request) error {
	if f == nil {
		return nil
	}
	return f(req)
}

func setupBodyContent(c *Client) MiddlewareFunc {
	return func(r *http.Request) error {
		if len(c.bodyForm) > 0 {
			c.ensureBodyContent()
		}
		if c.BodyContent != nil {
			for _, k := range c.bodyForm {
				err := c.BodyContent.Set(k.Name, k.Name)
				if err != nil {
					return err
				}
			}
			if c.Request.Header.Get("Content-Type") == "" {
				if ct := c.BodyContent.ContentType(); ct != "" {
					c.Request.Header.Set("Content-Type", ct)
				}
			}
			c.Request.Body = wrapReader(c.BodyContent.Read())
		}
		return nil
	}
}

func processAuth(c *Client) MiddlewareFunc {
	return func(r *http.Request) error {
		err := c.applyAuth()
		if err != nil {
			return err
		}
		return c.loadClientTLSCreds()
	}
}
