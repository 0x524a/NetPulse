package reporter

import (
	"fmt"
	"os"
)

// Reporter handles output of speed test results
type Reporter struct{}

// New creates a new Reporter instance
func New() *Reporter {
	return &Reporter{}
}

// PrintToConsole prints the results to the console
func (r *Reporter) PrintToConsole(results string) {
	fmt.Println(results)
}

// SaveToFile saves the results to a file
func (r *Reporter) SaveToFile(filename string, results string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	_, err = file.WriteString(results)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	fmt.Printf("Results saved to: %s\n", filename)
	return nil
}
