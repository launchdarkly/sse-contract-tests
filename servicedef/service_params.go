package servicedef

import "gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"

const CommandRestart = "restart"

type CreateStreamParams struct {
	Tag            string              `json:"tag"`
	CallbackURL    string              `json:"callbackUrl"`
	StreamURL      string              `json:"streamUrl"`
	InitialDelayMS ldvalue.OptionalInt `json:"initialDelayMs,omitempty"`
	LastEventID    string              `json:"lastEventId,omitempty"`
	Method         string              `json:"method,omitempty"`
	Body           string              `json:"body,omitempty"`
	Headers        map[string]string   `json:"headers,omitempty"`
	ReadTimeoutMS  ldvalue.OptionalInt `json:"readTimeoutMs,omitempty"`
}

type CommandParams struct {
	Command string        `json:"command"`
	Listen  *ListenParams `json:"listen"`
}

type ListenParams struct {
	Type string `json:"type"`
}
