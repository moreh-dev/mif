# E2E Test Structure

This directory contains end-to-end tests for MIF (Moai Inference Framework).

## File Organization

### Test Entry Points
- `e2e_suite_test.go` - Main test suite entry point (TestE2E, BeforeSuite, AfterSuite)
- `e2e_test.go` - Actual test cases

### Suite Management
- `suite_helpers.go` - Helper functions (setupInterruptHandler, cleanupKindCluster, cleanupMIFNamespace, cleanupE2ETempFiles, checkPrerequisites)
- `suite_setup.go` - Setup functions (setupKindCluster, setupPrerequisites, detectComponentState, setupMIF, setupPreset, setupGateway, etc.)
- `suite_wait.go` - Wait functions (waitForMIFComponents, ensureECRTokenRefresherSecret)

### Configuration
- `config.go` - Test configuration structure and initialization
- `constants.go` - Test constants (timeouts, resource names, etc.)
- `env_vars.go` - Environment variable definitions
- `env_vars_doc.go` - Environment variable documentation

### Values Files
- `values_files.go` - Helm values file creation functions

### Utilities
- `cmd/printenv/` - Print environment utility

## Note

All files in this directory are part of the same `package e2e` and must remain in the same directory due to Go's package structure requirements. Files are organized by functionality using naming conventions (prefixes like `suite_`, `config`, `values_`).
