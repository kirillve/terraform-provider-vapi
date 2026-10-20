package vapi

import (
	"errors"
	"net/http"
	"testing"
)

func addJSONSeeds(f *testing.F) {
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{"id":"f1","bytes":12}`))
	f.Add([]byte(`{"id":"f1","bytes":"12"}`))
	f.Add([]byte(`{"bytes":true}`))
	f.Add([]byte(`{"type":"object","properties":{"a":{"type":"string","enum":["x"]}},"required":["a"]}`))
	f.Add([]byte(`{"type":["string","null"],"properties":{"a":{"enum":[1,2]}}}`))
	f.Add([]byte(`[1,2,3]`))
	f.Add([]byte(`{"`))
}

// FuzzFileResponseUnmarshal feeds arbitrary bytes to the custom FileResponse decoder.
func FuzzFileResponseUnmarshal(f *testing.F) {
	addJSONSeeds(f)
	f.Fuzz(func(_ *testing.T, data []byte) {
		var fr FileResponse
		_ = fr.UnmarshalJSON(data)
	})
}

// FuzzResponseFunctionParamsUnmarshal checks that the decoder never fails and keeps the raw schema.
func FuzzResponseFunctionParamsUnmarshal(f *testing.F) {
	addJSONSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		var p ResponseFunctionParams
		if err := p.UnmarshalJSON(data); err != nil {
			t.Fatalf("UnmarshalJSON must not fail: %v", err)
		}
		if string(p.Raw) != string(data) {
			t.Fatalf("Raw = %q, want %q", p.Raw, data)
		}
	})
}

// FuzzDoDecode feeds arbitrary 2xx response bodies through Do into every response DTO.
func FuzzDoDecode(f *testing.F) {
	addJSONSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		client := newTestClient(roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, string(data)), nil
		}))
		outs := []any{
			&Assistant{}, &TwilioPhoneNumber{}, &SIPTrunk{}, &FileResponse{},
			&ToolFunctionResponse{}, &ToolQueryFunctionResponse{}, &ImportSIPTrunkPhoneNumberResponse{},
		}
		for _, out := range outs {
			err := client.Do(http.MethodGet, "x", nil, out)
			var decodeErr *DecodeError
			if err != nil && !errors.As(err, &decodeErr) {
				t.Fatalf("Do(%T) returned %T, want nil or *DecodeError", out, err)
			}
		}
	})
}
