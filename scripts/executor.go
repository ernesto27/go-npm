package scripts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type ScriptExecutor struct {
	nodeModulesPath string
	timeout         time.Duration
}

func NewScriptExecutor(nodeModulesPath string) *ScriptExecutor {
	return &ScriptExecutor{
		nodeModulesPath: nodeModulesPath,
		timeout:         5 * time.Minute,
	}
}

func (se *ScriptExecutor) Execute(script, workDir, pkgName, pkgVersion, event string) error {
	if script == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), se.timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/c", script)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", script)
	}

	cmd.Dir = workDir
	cmd.Env = se.buildEnvironment(pkgName, pkgVersion, event)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("$ %s\n", script)

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("script timeout after %v: %w", se.timeout, err)
		}
		return fmt.Errorf("script execution failed: %w", err)
	}

	return nil
}

func (se *ScriptExecutor) buildEnvironment(pkgName, pkgVersion, event string) []string {
	env := os.Environ()

	env = append(env,
		"npm_lifecycle_event="+event,
		"npm_package_name="+pkgName,
		"npm_package_version="+pkgVersion,
	)

	binPath := filepath.Join(se.nodeModulesPath, ".bin")
	path := os.Getenv("PATH")

	pathSeparator := ":"
	if runtime.GOOS == "windows" {
		pathSeparator = ";"
	}

	newPath := binPath + pathSeparator + path
	env = se.setEnv(env, "PATH", newPath)

	return env
}

func (se *ScriptExecutor) setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}
