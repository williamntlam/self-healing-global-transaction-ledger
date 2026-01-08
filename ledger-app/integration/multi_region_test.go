package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/project-atlas/ledger-app/internal/models"
)

// Test configuration - adjust these to match your setup
const (
	USEndpoint = "http://localhost:8080" // US region endpoint
	EUEndpoint = "http://localhost:8081" // EU region endpoint
	GlobalLBEndpoint = "http://localhost:8082" // Global load balancer endpoint
	TestTimeout = 30 * time.Second
)

// TransactionRequest represents a transaction creation request
type TransactionRequest struct {
	FromAccount string `json:"from_account"`
	ToAccount   string `json:"to_account"`
	Amount      string `json:"amount"`
}

// TransactionResponse represents the API response
type TransactionResponse struct {
	Transaction *models.Transaction `json:"transaction,omitempty"`
	Message     string               `json:"message,omitempty"`
	Error       string               `json:"error,omitempty"`
}

// TestMultiRegionConsistency tests that transactions are consistent across regions
func TestMultiRegionConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("both regions up - create and verify consistency", func(t *testing.T) {
		// Create transaction via US region
		txID := createTransaction(t, USEndpoint, "account-1", "account-2", "100.50")
		
		// Verify transaction exists in US region
		txUS := getTransaction(t, USEndpoint, txID)
		if txUS == nil {
			t.Fatal("Transaction not found in US region")
		}

		// Wait for replication (CockroachDB should replicate quickly)
		time.Sleep(2 * time.Second)

		// Verify transaction exists in EU region (via direct query)
		txEU := getTransaction(t, EUEndpoint, txID)
		if txEU == nil {
			t.Fatal("Transaction not found in EU region - replication failed")
		}

		// Verify data consistency
		if txUS.ID != txEU.ID {
			t.Errorf("Transaction ID mismatch: US=%s, EU=%s", txUS.ID, txEU.ID)
		}
		if txUS.Amount.String() != txEU.Amount.String() {
			t.Errorf("Amount mismatch: US=%s, EU=%s", txUS.Amount.String(), txEU.Amount.String())
		}
		if txUS.FromAccount != txEU.FromAccount {
			t.Errorf("FromAccount mismatch: US=%s, EU=%s", txUS.FromAccount, txEU.FromAccount)
		}
		if txUS.ToAccount != txEU.ToAccount {
			t.Errorf("ToAccount mismatch: US=%s, EU=%s", txUS.ToAccount, txEU.ToAccount)
		}
	})

	t.Run("both regions up - update and verify consistency", func(t *testing.T) {
		// Create transaction
		txID := createTransaction(t, USEndpoint, "account-3", "account-4", "200.75")
		
		// Update status via US region
		updateTransactionStatus(t, USEndpoint, txID, "completed")
		
		// Wait for replication
		time.Sleep(2 * time.Second)
		
		// Verify status updated in both regions
		txUS := getTransaction(t, USEndpoint, txID)
		txEU := getTransaction(t, EUEndpoint, txID)
		
		if txUS.Status != "completed" {
			t.Errorf("US region status not updated: got %s, want completed", txUS.Status)
		}
		if txEU.Status != "completed" {
			t.Errorf("EU region status not updated: got %s, want completed", txEU.Status)
		}
	})

	t.Run("both regions up - list transactions consistency", func(t *testing.T) {
		// Create multiple transactions
		txIDs := make([]uuid.UUID, 3)
		for i := 0; i < 3; i++ {
			txIDs[i] = createTransaction(t, USEndpoint, 
				fmt.Sprintf("account-%d", i), 
				fmt.Sprintf("account-%d", i+10), 
				fmt.Sprintf("%d.50", 100+i))
		}
		
		// Wait for replication
		time.Sleep(2 * time.Second)
		
		// List transactions from both regions
		usList := listTransactions(t, USEndpoint)
		euList := listTransactions(t, EUEndpoint)
		
		// Verify both regions have the same transactions
		if len(usList) != len(euList) {
			t.Errorf("Transaction count mismatch: US=%d, EU=%d", len(usList), len(euList))
		}
		
		// Verify all created transactions are in both lists
		for _, expectedID := range txIDs {
			foundUS := false
			foundEU := false
			for _, tx := range usList {
				if tx.ID == expectedID {
					foundUS = true
					break
				}
			}
			for _, tx := range euList {
				if tx.ID == expectedID {
					foundEU = true
					break
				}
			}
			if !foundUS {
				t.Errorf("Transaction %s not found in US region list", expectedID)
			}
			if !foundEU {
				t.Errorf("Transaction %s not found in EU region list", expectedID)
			}
		}
	})
}

// TestFailoverWithOneRegionDown tests failover when one region is down
func TestFailoverWithOneRegionDown(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("US region down - create via EU", func(t *testing.T) {
		// This test assumes you've manually stopped the US region
		// or use the blast_radius.sh script in pause mode
		
		// Try to create transaction via US (should fail)
		_, err := tryCreateTransaction(USEndpoint, "account-1", "account-2", "100.50")
		if err == nil {
			t.Log("US region still up - skipping failover test")
			return
		}
		
		// Create transaction via EU (should succeed)
		txID := createTransaction(t, EUEndpoint, "account-1", "account-2", "100.50")
		
		// Verify transaction exists in EU
		txEU := getTransaction(t, EUEndpoint, txID)
		if txEU == nil {
			t.Fatal("Transaction not found in EU region")
		}
		
		// When US comes back, verify transaction is replicated
		// (This would require waiting for US to come back up)
		t.Logf("Transaction %s created successfully via EU during US outage", txID)
	})

	t.Run("EU region down - create via US", func(t *testing.T) {
		// Try to create transaction via EU (should fail)
		_, err := tryCreateTransaction(EUEndpoint, "account-3", "account-4", "200.75")
		if err == nil {
			t.Log("EU region still up - skipping failover test")
			return
		}
		
		// Create transaction via US (should succeed)
		txID := createTransaction(t, USEndpoint, "account-3", "account-4", "200.75")
		
		// Verify transaction exists in US
		txUS := getTransaction(t, USEndpoint, txID)
		if txUS == nil {
			t.Fatal("Transaction not found in US region")
		}
		
		t.Logf("Transaction %s created successfully via US during EU outage", txID)
	})
}

// TestLoadBalancerFailover tests automatic failover via global load balancer
func TestLoadBalancerFailover(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("load balancer routes to healthy region", func(t *testing.T) {
		// Create transaction via load balancer
		txID := createTransaction(t, GlobalLBEndpoint, "account-1", "account-2", "100.50")
		
		// Verify transaction exists (check both regions)
		txUS := getTransaction(t, USEndpoint, txID)
		txEU := getTransaction(t, EUEndpoint, txID)
		
		// At least one region should have the transaction
		if txUS == nil && txEU == nil {
			t.Fatal("Transaction not found in any region")
		}
	})

	t.Run("load balancer fails over when one region down", func(t *testing.T) {
		// This test requires one region to be down
		// You can use: kubectl scale deployment ledger-app --replicas=0 -n <namespace> --context k3d-dc-us
		
		// Try to create transaction via load balancer
		// It should automatically route to the healthy region
		txID, err := tryCreateTransaction(GlobalLBEndpoint, "account-1", "account-2", "100.50")
		if err != nil {
			t.Fatalf("Load balancer failed to route to healthy region: %v", err)
		}
		
		// Verify transaction was created in the surviving region
		// Check which region is up
		txUS := getTransaction(t, USEndpoint, txID)
		txEU := getTransaction(t, EUEndpoint, txID)
		
		if txUS == nil && txEU == nil {
			t.Fatal("Transaction not found in any region after failover")
		}
		
		t.Logf("Transaction %s created successfully via load balancer during failover", txID)
	})

	t.Run("load balancer health check", func(t *testing.T) {
		// Check load balancer health endpoint
		resp, err := http.Get(GlobalLBEndpoint + "/health")
		if err != nil {
			t.Fatalf("Failed to check load balancer health: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Load balancer health check failed: status %d", resp.StatusCode)
		}
	})
}

// Helper functions

func createTransaction(t *testing.T, endpoint, from, to, amount string) uuid.UUID {
	txID, err := tryCreateTransaction(endpoint, from, to, amount)
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}
	return txID
}

func tryCreateTransaction(endpoint, from, to, amount string) (uuid.UUID, error) {
	reqBody := TransactionRequest{
		FromAccount: from,
		ToAccount:   to,
		Amount:      amount,
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return uuid.Nil, err
	}
	
	client := &http.Client{Timeout: TestTimeout}
	resp, err := client.Post(endpoint+"/transactions", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return uuid.Nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		return uuid.Nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var txResp TransactionResponse
	if err := json.NewDecoder(resp.Body).Decode(&txResp); err != nil {
		return uuid.Nil, err
	}
	
	if txResp.Transaction == nil {
		return uuid.Nil, fmt.Errorf("transaction is nil in response")
	}
	
	return txResp.Transaction.ID, nil
}

func getTransaction(t *testing.T, endpoint string, txID uuid.UUID) *models.Transaction {
	client := &http.Client{Timeout: TestTimeout}
	resp, err := client.Get(endpoint + "/transactions/" + txID.String())
	if err != nil {
		t.Fatalf("Failed to get transaction: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Unexpected status code: %d", resp.StatusCode)
	}
	
	var txResp TransactionResponse
	if err := json.NewDecoder(resp.Body).Decode(&txResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	return txResp.Transaction
}

func updateTransactionStatus(t *testing.T, endpoint string, txID uuid.UUID, status string) {
	// This assumes you have a PATCH or PUT endpoint for updating status
	// If not, you'll need to add it to the API
	// For now, this is a placeholder
	t.Logf("Updating transaction %s status to %s (endpoint not yet implemented)", txID, status)
}

func listTransactions(t *testing.T, endpoint string) []*models.Transaction {
	client := &http.Client{Timeout: TestTimeout}
	resp, err := client.Get(endpoint + "/transactions?limit=100")
	if err != nil {
		t.Fatalf("Failed to list transactions: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Unexpected status code: %d", resp.StatusCode)
	}
	
	var result struct {
		Transactions []*models.Transaction `json:"transactions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	return result.Transactions
}
