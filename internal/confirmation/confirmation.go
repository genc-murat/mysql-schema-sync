package confirmation

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"mysql-schema-sync/internal/migration"
	"mysql-schema-sync/internal/schema"
)

// ConfirmationService handles user confirmation for schema changes
type ConfirmationService interface {
	ConfirmChanges(diff *schema.SchemaDiff, plan *migration.MigrationPlan, autoApprove bool) (bool, error)
	DisplayChangeSummary(diff *schema.SchemaDiff, plan *migration.MigrationPlan) error
	HandleInterruption() error
}

// confirmationService implements the ConfirmationService interface
type confirmationService struct {
	formatter *schema.DisplayFormatter
	reader    *bufio.Reader
}

// NewConfirmationService creates a new ConfirmationService instance
func NewConfirmationService(useColors bool) ConfirmationService {
	return &confirmationService{
		formatter: schema.NewDisplayFormatter(true, useColors),
		reader:    bufio.NewReader(os.Stdin),
	}
}

// ConfirmChanges displays changes and prompts for user confirmation
func (cs *confirmationService) ConfirmChanges(diff *schema.SchemaDiff, plan *migration.MigrationPlan, autoApprove bool) (bool, error) {
	// Check if there are no changes
	if cs.formatter.IsEmpty(diff) {
		fmt.Println(cs.formatter.FormatCompactSummary(diff))
		return false, nil // No changes to apply
	}

	// Display the change summary
	if err := cs.DisplayChangeSummary(diff, plan); err != nil {
		return false, fmt.Errorf("failed to display change summary: %w", err)
	}

	// Auto-approve if requested
	if autoApprove {
		fmt.Println("\n" + cs.formatter.Colorize("✓ Auto-approving changes...", "green"))
		return true, nil
	}

	// Set up interrupt handling
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)

	// Create a channel for user input
	inputChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	// Start goroutine to read user input
	go func() {
		input, err := cs.promptForConfirmation()
		if err != nil {
			errorChan <- err
			return
		}
		inputChan <- input
	}()

	// Wait for either user input or interrupt
	select {
	case <-interruptChan:
		fmt.Println("\n" + cs.formatter.Colorize("⚠ Operation cancelled by user", "yellow"))
		return false, cs.HandleInterruption()
	case err := <-errorChan:
		return false, fmt.Errorf("failed to read user input: %w", err)
	case input := <-inputChan:
		return cs.parseConfirmationInput(input), nil
	}
}

// DisplayChangeSummary displays a detailed summary of changes and warnings
func (cs *confirmationService) DisplayChangeSummary(diff *schema.SchemaDiff, plan *migration.MigrationPlan) error {
	// Display formatted schema differences
	fmt.Println(cs.formatter.FormatSchemaDiff(diff))

	// Display warnings if any
	if len(plan.Warnings) > 0 {
		fmt.Println(cs.formatter.Colorize("⚠ WARNINGS", "yellow"))
		fmt.Println(strings.Repeat("=", 50))
		for i, warning := range plan.Warnings {
			fmt.Printf("%d. %s\n", i+1, cs.formatter.Colorize(warning, "yellow"))
		}
		fmt.Println()
	}

	// Display destructive operations warning
	destructiveCount := cs.countDestructiveOperations(plan)
	if destructiveCount > 0 {
		fmt.Println(cs.formatter.Colorize("🚨 DESTRUCTIVE OPERATIONS DETECTED", "red"))
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("This migration contains %d potentially destructive operation(s) that may result in data loss.\n", destructiveCount)
		fmt.Println("Please review the changes carefully before proceeding.")
		fmt.Println()
	}

	// Display execution summary
	fmt.Println(cs.formatter.Colorize("Execution Summary", "bold"))
	fmt.Println(strings.Repeat("-", 30))
	fmt.Printf("Total statements to execute: %d\n", len(plan.Statements))
	fmt.Printf("Destructive operations: %d\n", destructiveCount)
	fmt.Printf("Estimated execution time: %s\n", cs.estimateExecutionTime(plan))
	fmt.Println()

	return nil
}

// promptForConfirmation prompts the user for confirmation
func (cs *confirmationService) promptForConfirmation() (string, error) {
	fmt.Print(cs.formatter.Colorize("Do you want to apply these changes? [y/N/d]: ", "bold"))

	input, err := cs.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	return strings.TrimSpace(input), nil
}

// parseConfirmationInput parses the user's confirmation input
func (cs *confirmationService) parseConfirmationInput(input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))

	switch input {
	case "y", "yes":
		return true
	case "d", "details":
		// Show detailed SQL statements
		fmt.Println("\n" + cs.formatter.Colorize("SQL Statements to be executed:", "bold"))
		fmt.Println(strings.Repeat("=", 50))
		// This would need access to the plan, so we'll handle this in the main confirmation flow
		return false
	case "n", "no", "":
		return false
	default:
		fmt.Printf("Invalid input '%s'. Please enter 'y' for yes, 'n' for no, or 'd' for details.\n", input)
		// Recursively prompt again
		input, err := cs.promptForConfirmation()
		if err != nil {
			return false
		}
		return cs.parseConfirmationInput(input)
	}
}

// HandleInterruption handles graceful shutdown on user interruption
func (cs *confirmationService) HandleInterruption() error {
	fmt.Println("Cleaning up...")
	// Perform any necessary cleanup here
	return nil
}

// countDestructiveOperations counts the number of destructive operations in the plan
func (cs *confirmationService) countDestructiveOperations(plan *migration.MigrationPlan) int {
	count := 0
	for _, stmt := range plan.Statements {
		if stmt.IsDestructive {
			count++
		}
	}
	return count
}

// estimateExecutionTime provides a rough estimate of execution time
func (cs *confirmationService) estimateExecutionTime(plan *migration.MigrationPlan) string {
	// Simple heuristic based on statement count and types
	statementCount := len(plan.Statements)

	if statementCount == 0 {
		return "< 1 second"
	} else if statementCount < 10 {
		return "< 5 seconds"
	} else if statementCount < 50 {
		return "< 30 seconds"
	} else {
		return "> 1 minute"
	}
}

// ConfirmWithDetails provides an enhanced confirmation flow with detailed SQL display
func (cs *confirmationService) ConfirmWithDetails(diff *schema.SchemaDiff, plan *migration.MigrationPlan, autoApprove bool) (bool, error) {
	for {
		// Display the change summary
		if err := cs.DisplayChangeSummary(diff, plan); err != nil {
			return false, fmt.Errorf("failed to display change summary: %w", err)
		}

		// Auto-approve if requested
		if autoApprove {
			fmt.Println("\n" + cs.formatter.Colorize("✓ Auto-approving changes...", "green"))
			return true, nil
		}

		// Set up interrupt handling
		interruptChan := make(chan os.Signal, 1)
		signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)

		// Create channels for user input
		inputChan := make(chan string, 1)
		errorChan := make(chan error, 1)

		// Start goroutine to read user input
		go func() {
			input, err := cs.promptForConfirmation()
			if err != nil {
				errorChan <- err
				return
			}
			inputChan <- input
		}()

		// Wait for either user input or interrupt
		select {
		case <-interruptChan:
			fmt.Println("\n" + cs.formatter.Colorize("⚠ Operation cancelled by user", "yellow"))
			return false, cs.HandleInterruption()
		case err := <-errorChan:
			return false, fmt.Errorf("failed to read user input: %w", err)
		case input := <-inputChan:
			input = strings.ToLower(strings.TrimSpace(input))

			switch input {
			case "y", "yes":
				return true, nil
			case "n", "no", "":
				fmt.Println(cs.formatter.Colorize("✓ Operation cancelled by user", "green"))
				return false, nil
			case "d", "details":
				cs.displaySQLDetails(plan)
				continue // Loop back to prompt again
			default:
				fmt.Printf("Invalid input '%s'. Please enter 'y' for yes, 'n' for no, or 'd' for details.\n", input)
				continue // Loop back to prompt again
			}
		}
	}
}

// displaySQLDetails shows the detailed SQL statements that will be executed
func (cs *confirmationService) displaySQLDetails(plan *migration.MigrationPlan) {
	fmt.Println("\n" + cs.formatter.Colorize("SQL Statements to be executed:", "bold"))
	fmt.Println(strings.Repeat("=", 60))

	for i, stmt := range plan.Statements {
		// Display statement number and description
		fmt.Printf("\n%s %d. %s\n",
			cs.formatter.Colorize("Statement", "bold"),
			i+1,
			stmt.Description)

		// Mark destructive operations
		if stmt.IsDestructive {
			fmt.Printf("   %s\n", cs.formatter.Colorize("⚠ DESTRUCTIVE OPERATION", "red"))
		}

		// Display the SQL
		fmt.Printf("   SQL: %s\n", cs.formatter.Colorize(stmt.SQL, "blue"))
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
}
