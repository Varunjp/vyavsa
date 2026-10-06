package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type TestUserData struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	Role        string `json:"role"`
	AccessToken string `json:"access_token,omitempty"`
}

type TestTenantData struct {
	TenantID         string         `json:"tenant_id"`
	Name             string         `json:"name"`
	Email            string         `json:"email"`
	Admin            TestUserData   `json:"admin"`
	Users            []TestUserData `json:"users"`
	SampleCustomerID string         `json:"sample_customer_id,omitempty"`
	SampleBankID     string         `json:"sample_bank_id,omitempty"`
	SampleEmployeeID string         `json:"sample_employee_id,omitempty"`
}

type TestDataset struct {
	CreatedAt   time.Time        `json:"created_at"`
	TenantCount int              `json:"tenant_count"`
	TotalUsers  int              `json:"total_users"`
	Password    string           `json:"password"`
	Tenants     []TestTenantData `json:"tenants"`
}

func main() {
	var (
		action          string
		tenantCount     int
		usersPerTenant  int
		outputPath      string
		defaultPassword string
		enableLoadTest  bool
	)

	flag.StringVar(&action, "action", "setup", "Action to perform: setup, teardown, status")
	flag.IntVar(&tenantCount, "tenants", 25, "Number of synthetic tenants to create")
	flag.IntVar(&usersPerTenant, "users-per-tenant", 2, "Number of users per tenant (including admin)")
	flag.StringVar(&outputPath, "output", "load-tests/data/test-users.json", "Path to write test dataset JSON")
	flag.StringVar(&defaultPassword, "password", "TestPass@12345", "Default password for synthetic test accounts")
	flag.BoolVar(&enableLoadTest, "enable-load-test", false, "Explicit confirmation to enable load test modifications")
	flag.Parse()

	// 1. Safety verification
	if !enableLoadTest && strings.ToLower(os.Getenv("LOAD_TEST_ENABLED")) != "true" {
		fmt.Fprintf(os.Stderr, "FATAL: Load testing is disabled for safety. Run with -enable-load-test or set LOAD_TEST_ENABLED=true\n")
		os.Exit(1)
	}

	appEnv := strings.ToLower(os.Getenv("APP_ENV"))
	if appEnv == "production" || strings.ToLower(os.Getenv("ENVIRONMENT")) == "production" {
		fmt.Fprintf(os.Stderr, "FATAL: Load test data generator refuses to run in production environment (APP_ENV=%s)!\n", appEnv)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx := context.Background()

	pg, err := database.NewPostgres(ctx, cfg.Database, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pg.Close()

	loadJwtCfg := cfg.JWT
	loadJwtCfg.AccessExpiry = 24 * time.Hour
	jwtManager := auth.NewJWTManager(loadJwtCfg)

	switch action {
	case "setup":
		if err := runSetup(ctx, pg, jwtManager, tenantCount, usersPerTenant, defaultPassword, outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
			os.Exit(1)
		}
	case "teardown":
		if err := runTeardown(ctx, pg, outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "Teardown failed: %v\n", err)
			os.Exit(1)
		}
	case "status":
		if err := runStatus(ctx, pg); err != nil {
			fmt.Fprintf(os.Stderr, "Status check failed: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown action: %s. Use setup, teardown, or status.\n", action)
		os.Exit(1)
	}
}

func runSetup(
	ctx context.Context,
	pg *database.Postgres,
	jwtManager auth.JWTManager,
	tenantCount int,
	usersPerTenant int,
	defaultPassword string,
	outputPath string,
) error {
	fmt.Printf("[Setup] Preparing %d synthetic tenants with %d users each...\n", tenantCount, usersPerTenant)

	// Clean up any prior test tenants to avoid unique constraint conflicts
	_, _ = cleanSyntheticData(ctx, pg)

	// Pre-hash password once using bcrypt (cost 10 for fast test generation, but realistic bcrypt hash)
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), 10)
	if err != nil {
		return fmt.Errorf("failed to hash test password: %w", err)
	}
	passwordHash := string(hashedBytes)

	// Find active subscription plan
	var planID uuid.UUID
	var planName string
	err = pg.Pool.QueryRow(ctx, `
		SELECT id, plan_name 
		FROM platform_plans 
		WHERE status = 'active' 
		ORDER BY price ASC 
		LIMIT 1
	`).Scan(&planID, &planName)
	if err != nil {
		return fmt.Errorf("failed to query active subscription plan: %w", err)
	}

	dataset := TestDataset{
		CreatedAt:   time.Now().UTC(),
		TenantCount: tenantCount,
		Password:    defaultPassword,
		Tenants:     make([]TestTenantData, 0, tenantCount),
	}

	totalUsers := 0

	for i := 1; i <= tenantCount; i++ {
		tenantID := uuid.New()
		tenantName := fmt.Sprintf("LoadTest Tenant %03d", i)
		tenantEmail := fmt.Sprintf("loadtest_tenant_%03d@vyavsa-test.internal", i)
		tenantPhone := fmt.Sprintf("+919000000%03d", i)

		// 1. Insert Tenant
		_, err := pg.Pool.Exec(ctx, `
			INSERT INTO tenants (id, name, email, phone, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 'active', NOW(), NOW())
		`, tenantID, tenantName, tenantEmail, tenantPhone)
		if err != nil {
			return fmt.Errorf("failed to insert tenant %d: %w", i, err)
		}

		// 2. Insert Active Subscription
		subID := uuid.New()
		endDate := time.Now().AddDate(1, 0, 0)
		_, err = pg.Pool.Exec(ctx, `
			INSERT INTO platform_subscriptions (id, tenant_id, current_plan_id, current_plan_name, status, start_date, end_date, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 'active', NOW(), $5, NOW(), NOW())
		`, subID, tenantID, planID, planName, endDate)
		if err != nil {
			return fmt.Errorf("failed to insert subscription for tenant %d: %w", i, err)
		}

		// 3. Insert Initial Financial Summary
		_, err = pg.Pool.Exec(ctx, `
			INSERT INTO tenant_financial_summary (id, tenant_id, cash_balance, bank_balance, total_receivable, total_payable, updated_at)
			VALUES ($1, $2, 100000.00, 500000.00, 50000.00, 20000.00, NOW())
		`, uuid.New(), tenantID)
		if err != nil {
			return fmt.Errorf("failed to insert financial summary for tenant %d: %w", i, err)
		}

		// 4. Insert Sample Bank
		bankID := uuid.New()
		_, err = pg.Pool.Exec(ctx, `
			INSERT INTO tenant_bank (id, tenant_id, bank_name, account_number, ifsc_or_routing, current_balance, status, created_at, updated_at)
			VALUES ($1, $2, 'HDFC Bank', $3, 'HDFC0001234', 500000.00, 'active', NOW(), NOW())
		`, bankID, tenantID, fmt.Sprintf("123456789%03d", i))
		if err != nil {
			return fmt.Errorf("failed to insert bank for tenant %d: %w", i, err)
		}

		// 5. Insert Sample Customer
		custID := uuid.New()
		_, err = pg.Pool.Exec(ctx, `
			INSERT INTO tenant_customer (id, tenant_id, customer_name, phone, opening_balance, current_balance, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 5000.00, 5000.00, 'active', NOW(), NOW())
		`, custID, tenantID, fmt.Sprintf("Customer Alpha %03d", i), fmt.Sprintf("+919111111%03d", i))
		if err != nil {
			return fmt.Errorf("failed to insert customer for tenant %d: %w", i, err)
		}

		// 6. Insert Sample Employee
		empID := uuid.New()
		_, err = pg.Pool.Exec(ctx, `
			INSERT INTO tenant_employees (id, tenant_id, name, phone, salary, ot_rate, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 30000.00, 150.00, 'active', NOW(), NOW())
		`, empID, tenantID, fmt.Sprintf("Employee One %03d", i), fmt.Sprintf("+919222222%03d", i))
		if err != nil {
			return fmt.Errorf("failed to insert employee for tenant %d: %w", i, err)
		}

		// 7. Insert Tenant Admin User
		adminID := uuid.New()
		adminEmail := fmt.Sprintf("loadtest_admin_%03d@vyavsa-test.internal", i)
		_, err = pg.Pool.Exec(ctx, `
			INSERT INTO tenant_user (id, tenant_id, name, role, email, password_hash, status, created_at, updated_at)
			VALUES ($1, $2, $3, 'admin', $4, $5, 'active', NOW(), NOW())
		`, adminID, tenantID, fmt.Sprintf("Admin %03d", i), adminEmail, passwordHash)
		if err != nil {
			return fmt.Errorf("failed to insert admin user for tenant %d: %w", i, err)
		}
		totalUsers++

		// Generate access token for admin
		adminTokens, _ := jwtManager.GenerateTokenPair(adminID, &tenantID, adminEmail, auth.RoleTenantAdmin, auth.UserTypeTenantUser)
		adminTokenStr := ""
		if adminTokens != nil {
			adminTokenStr = adminTokens.AccessToken
		}

		adminData := TestUserData{
			ID:          adminID.String(),
			Email:       adminEmail,
			Password:    defaultPassword,
			Role:        "admin",
			AccessToken: adminTokenStr,
		}

		// 8. Insert Additional Staff Users
		users := make([]TestUserData, 0, usersPerTenant-1)
		for u := 2; u <= usersPerTenant; u++ {
			userID := uuid.New()
			userEmail := fmt.Sprintf("loadtest_user_%03d_%d@vyavsa-test.internal", i, u)
			_, err = pg.Pool.Exec(ctx, `
				INSERT INTO tenant_user (id, tenant_id, name, role, email, password_hash, status, created_at, updated_at)
				VALUES ($1, $2, $3, 'user', $4, $5, 'active', NOW(), NOW())
			`, userID, tenantID, fmt.Sprintf("Staff %03d-%d", i, u), userEmail, passwordHash)
			if err != nil {
				return fmt.Errorf("failed to insert staff user %d for tenant %d: %w", u, i, err)
			}
			totalUsers++

			userTokens, _ := jwtManager.GenerateTokenPair(userID, &tenantID, userEmail, auth.RoleTenantUser, auth.UserTypeTenantUser)
			userTokenStr := ""
			if userTokens != nil {
				userTokenStr = userTokens.AccessToken
			}

			users = append(users, TestUserData{
				ID:          userID.String(),
				Email:       userEmail,
				Password:    defaultPassword,
				Role:        "user",
				AccessToken: userTokenStr,
			})
		}

		tenantEntry := TestTenantData{
			TenantID:         tenantID.String(),
			Name:             tenantName,
			Email:            tenantEmail,
			Admin:            adminData,
			Users:            users,
			SampleCustomerID: custID.String(),
			SampleBankID:     bankID.String(),
			SampleEmployeeID: empID.String(),
		}
		dataset.Tenants = append(dataset.Tenants, tenantEntry)
	}

	dataset.TotalUsers = totalUsers

	// Write dataset to output file
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for dataset: %w", err)
	}

	fileBytes, err := json.MarshalIndent(dataset, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dataset to JSON: %w", err)
	}

	if err := os.WriteFile(outputPath, fileBytes, 0644); err != nil {
		return fmt.Errorf("failed to write dataset file: %w", err)
	}

	fmt.Printf("[Setup Complete] Successfully created %d synthetic tenants, %d users.\n", tenantCount, totalUsers)
	fmt.Printf("[Dataset Saved] %s (Size: %d bytes)\n", outputPath, len(fileBytes))
	return nil
}

func runTeardown(ctx context.Context, pg *database.Postgres, outputPath string) error {
	fmt.Println("[Teardown] Cleaning up all synthetic load test data...")
	deleted, err := cleanSyntheticData(ctx, pg)
	if err != nil {
		return err
	}

	if outputPath != "" {
		_ = os.Remove(outputPath)
	}

	fmt.Printf("[Teardown Complete] Removed %d synthetic tenants and all cascaded records.\n", deleted)
	return nil
}

func cleanSyntheticData(ctx context.Context, pg *database.Postgres) (int64, error) {
	tag, err := pg.Pool.Exec(ctx, `
		DELETE FROM tenants 
		WHERE email LIKE 'loadtest_%@vyavsa-test.internal' 
		   OR name LIKE 'LoadTest Tenant%'
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to delete synthetic tenants: %w", err)
	}
	return tag.RowsAffected(), nil
}

func runStatus(ctx context.Context, pg *database.Postgres) error {
	var count int
	err := pg.Pool.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM tenants 
		WHERE email LIKE 'loadtest_%@vyavsa-test.internal'
	`).Scan(&count)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("failed to count synthetic tenants: %w", err)
	}

	var userCount int
	err = pg.Pool.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM tenant_user 
		WHERE email LIKE 'loadtest_%@vyavsa-test.internal'
	`).Scan(&userCount)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("failed to count synthetic users: %w", err)
	}

	fmt.Printf("[Status] Synthetic test tenants: %d | Synthetic test users: %d\n", count, userCount)
	return nil
}
