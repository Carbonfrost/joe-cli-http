// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpserver // intentional

import ()

type ServerAttributes struct {
	AccessLog string
}

func Attributes(s *Server) *ServerAttributes {
	return &ServerAttributes{
		AccessLog: s.accessLog,
	}
}
