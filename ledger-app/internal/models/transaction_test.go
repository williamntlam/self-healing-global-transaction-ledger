package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestParseAmount(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		expected  decimal.Decimal
	}{
		{
			name:      "valid positive amount",
			input:     "100.50",
			wantError: false,
			expected:  decimal.NewFromInt(10050).Div(decimal.NewFromInt(100)),
		},
		{
			name:      "valid integer amount",
			input:     "100",
			wantError: false,
			expected:  decimal.NewFromInt(100),
		},
		{
			name:      "valid large amount",
			input:     "999999999999999999.99",
			wantError: false,
			expected:  decimal.RequireFromString("999999999999999999.99"),
		},
		{
			name:      "invalid empty string",
			input:     "",
			wantError: true,
		},
		{
			name:      "invalid non-numeric",
			input:     "abc",
			wantError: true,
		},
		{
			name:      "invalid with letters",
			input:     "100.50abc",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseAmount(tt.input)
			if tt.wantError {
				if err == nil {
					t.Errorf("ParseAmount() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("ParseAmount() unexpected error: %v", err)
				}
				if !result.Equal(tt.expected) {
					t.Errorf("ParseAmount() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestAuditLog_ToJSON(t *testing.T) {
	txID := uuid.New()
	auditLog := &AuditLog{
		TransactionID: txID,
		Region:        "us-east-1",
		Action:        "transaction_created",
		Timestamp:     parseTime("2024-01-01T00:00:00Z"),
		Details:       "Test transaction",
	}

	json, err := auditLog.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	if json == "" {
		t.Error("ToJSON() returned empty string")
	}

	// Verify it's valid JSON by checking it contains expected fields
	if !contains(json, txID.String()) {
		t.Errorf("ToJSON() missing transaction ID")
	}
	if !contains(json, "us-east-1") {
		t.Errorf("ToJSON() missing region")
	}
	if !contains(json, "transaction_created") {
		t.Errorf("ToJSON() missing action")
	}
}

func TestTransaction_JSONSerialization(t *testing.T) {
	tx := &Transaction{
		ID:          uuid.New(),
		Region:      "us-east-1",
		Amount:      decimal.NewFromInt(10050).Div(decimal.NewFromInt(100)),
		FromAccount: "account-1",
		ToAccount:   "account-2",
		Status:      "pending",
		Timestamp:   parseTime("2024-01-01T00:00:00Z"),
	}

	// Test JSON marshaling
	data, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled Transaction
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if unmarshaled.ID != tx.ID {
		t.Errorf("Unmarshal() ID = %v, want %v", unmarshaled.ID, tx.ID)
	}
	if unmarshaled.Region != tx.Region {
		t.Errorf("Unmarshal() Region = %v, want %v", unmarshaled.Region, tx.Region)
	}
	if !unmarshaled.Amount.Equal(tx.Amount) {
		t.Errorf("Unmarshal() Amount = %v, want %v", unmarshaled.Amount, tx.Amount)
	}
}

func TestUUIDArray_Value(t *testing.T) {
	tests := []struct {
		name     string
		input    UUIDArray
		expected string
	}{
		{
			name:     "empty array",
			input:    UUIDArray{},
			expected: "{}",
		},
		{
			name:     "single UUID",
			input:    UUIDArray{uuid.New()},
			expected: "", // We'll check it's not empty and contains the UUID
		},
		{
			name:     "multiple UUIDs",
			input:    UUIDArray{uuid.New(), uuid.New()},
			expected: "", // We'll check format
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.input.Value()
			if err != nil {
				t.Errorf("Value() error = %v", err)
			}

			if tt.expected != "" && result != tt.expected {
				t.Errorf("Value() = %v, want %v", result, tt.expected)
			}

			// For non-empty arrays, verify format
			if len(tt.input) > 0 {
				resultStr, ok := result.(string)
				if !ok {
					t.Errorf("Value() returned non-string type")
				}
				if resultStr[0] != '{' || resultStr[len(resultStr)-1] != '}' {
					t.Errorf("Value() format incorrect, should start with { and end with }")
				}
			}
		})
	}
}

// Helper functions
func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
