package rpcapi

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestBusinessListRequestLimitUsesZeroValue(t *testing.T) {
	tests := []struct {
		name    string
		decode  func([]byte) (int, error)
		newZero func() any
	}{
		{
			name: "PetListRequest",
			decode: func(data []byte) (int, error) {
				var request PetListRequest
				if err := json.Unmarshal(data, &request); err != nil {
					return 0, err
				}
				return request.Limit, nil
			},
			newZero: func() any { return PetListRequest{} },
		},
		{
			name: "RewardListRequest",
			decode: func(data []byte) (int, error) {
				var request RewardListRequest
				if err := json.Unmarshal(data, &request); err != nil {
					return 0, err
				}
				return request.Limit, nil
			},
			newZero: func() any { return RewardListRequest{} },
		},
		{
			name: "WalletTransactionsListRequest",
			decode: func(data []byte) (int, error) {
				var request WalletTransactionsListRequest
				if err := json.Unmarshal(data, &request); err != nil {
					return 0, err
				}
				return request.Limit, nil
			},
			newZero: func() any { return WalletTransactionsListRequest{} },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, ok := reflect.TypeOf(tt.newZero()).FieldByName("Limit")
			if !ok {
				t.Fatalf("%s.Limit field missing", tt.name)
			}
			if field.Type.Kind() != reflect.Int {
				t.Fatalf("%s.Limit type = %s, want int", tt.name, field.Type)
			}

			limit, err := tt.decode([]byte(`{}`))
			if err != nil {
				t.Fatalf("decode empty request error = %v", err)
			}
			if limit != 0 {
				t.Fatalf("empty request limit = %d, want 0", limit)
			}

			limit, err = tt.decode([]byte(`{"limit":7}`))
			if err != nil {
				t.Fatalf("decode limit request error = %v", err)
			}
			if limit != 7 {
				t.Fatalf("decoded limit = %d, want 7", limit)
			}

			data, err := json.Marshal(tt.newZero())
			if err != nil {
				t.Fatalf("marshal zero request error = %v", err)
			}
			if string(data) != "{}" {
				t.Fatalf("zero request JSON = %s, want {}", data)
			}
		})
	}
}
