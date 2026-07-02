package verifier

import "net/http"

type Noop struct{}

func (*Noop) Verify(_ *http.Request, _ []byte) error { return nil }
