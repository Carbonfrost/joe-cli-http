package httpserver

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/Carbonfrost/joe-cli-http/httpclient"
	"github.com/Carbonfrost/joe-cli-http/httpclient/expr"
	"github.com/Carbonfrost/joe-cli/extensions/expr/expander"
)

func ExpandRequest(r *http.Request, ww wrapResponseWriter) expander.Interface {
	return expander.Compose(expander.Func(func(s string) any {
		switch s {
		case "bytesWritten":
			return ww.BytesWritten()
		case "method":
			return expr.HTTPMethod(r.Method)
		case "protocol":
			return r.Proto
		case "statusCode":
			return expr.HTTPStatus(ww.Status())
		case "status":
			return fmt.Sprint(ww.Status(), " ", http.StatusText(ww.Status()))
		case "urlPath":
			return r.URL.Path
		case "header":
			var buf bytes.Buffer
			ww.Header().Write(&buf)
			return buf.String()
		}
		return nil
	}), expander.Prefix("header", httpclient.ExpandHeader(ww.Header())))
}

func expandTiming(start, end time.Time) expander.Interface {
	return expander.Map(map[string]any{
		"duration": end.Sub(start),
		"end":      end,
		"start":    start,
	})
}
