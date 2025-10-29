// Package executor provides script execution functionality
package executor

import (
	"log/slog"
	"os"
	"os/exec"

	"git.asdf.cafe/abs3nt/wallhaven_dl/errors"
)

// ScriptExecutor handles script execution
type ScriptExecutor struct {
	logger *slog.Logger
}

// NewScriptExecutor creates a new script executor
func NewScriptExecutor(logger *slog.Logger) *ScriptExecutor {
	return &ScriptExecutor{
		logger: logger,
	}
}

// Execute runs a script with the given image path
func (s *ScriptExecutor) Execute(scriptPath, imagePath string) error {
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return errors.NewValidationError("scriptPath", scriptPath, "file does not exist")
	}

	s.logger.Info("Executing script", "script", scriptPath, "image", imagePath)

	cmd := exec.Command(scriptPath, imagePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// cmd.Env defaults to nil, which means inherit parent environment

	if err := cmd.Run(); err != nil {
		s.logger.Error("Script execution failed", "error", err, "script", scriptPath)
		return errors.ErrScriptExecution
	}

	s.logger.Info("Script executed successfully", "script", scriptPath)
	return nil
}