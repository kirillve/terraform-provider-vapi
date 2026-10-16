package vapi

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestTwilioSmsEnabledPayload(t *testing.T) {
	// Create and PATCH both use ImportTwilioRequest.
	for _, operation := range []string{"create", "update"} {
		for _, tc := range []struct {
			name  string
			value *bool
		}{
			{name: "omitted"}, {name: "false", value: new(bool)}, {name: "true", value: func() *bool { v := true; return &v }()},
		} {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				payload := ImportTwilioRequest{}
				field := reflect.ValueOf(&payload).Elem().FieldByName("SmsEnabled")
				if !field.IsValid() {
					t.Fatal("ImportTwilioRequest is missing SmsEnabled")
				}
				if field.Type() != reflect.TypeOf((*bool)(nil)) {
					t.Fatal("SmsEnabled must be *bool")
				}
				field.Set(reflect.ValueOf(tc.value))
				raw, err := json.Marshal(payload)
				if err != nil {
					t.Fatal(err)
				}
				var body map[string]json.RawMessage
				if err := json.Unmarshal(raw, &body); err != nil {
					t.Fatal(err)
				}
				got, exists := body["smsEnabled"]
				if tc.value == nil {
					if exists {
						t.Fatalf("smsEnabled must be omitted: %s", raw)
					}
					return
				}
				want := "false"
				if *tc.value {
					want = "true"
				}
				if string(got) != want {
					t.Fatalf("smsEnabled = %s, want %s", got, want)
				}
			})
		}
	}
}
