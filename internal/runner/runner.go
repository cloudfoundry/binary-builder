// Package runner provides an interface for executing system commands,
// with a real implementation for production and a fake for testing.
package runner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Runner is the interface for executing system commands.
// All packages that shell out accept this interface, enabling
// unit tests to inject FakeRunner without executing anything.
type Runner interface {
	// Run executes a command and returns an error if it fails.
	Run(name string, args ...string) error

	// RunWithEnv executes a command with additional environment variables.
	RunWithEnv(env map[string]string, name string, args ...string) error

	// RunInDir executes a command in the specified directory.
	RunInDir(dir string, name string, args ...string) error

	// RunInDirWithEnv executes a command in the specified directory with additional env vars.
	RunInDirWithEnv(dir string, env map[string]string, name string, args ...string) error

	// Output executes a command and returns its stdout.
	Output(name string, args ...string) (string, error)
}

// RealRunner executes commands on the real system.
type RealRunner struct{}

// Run executes a command, inheriting the current process's environment.
func (r *RealRunner) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running %s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

// RunWithEnv executes a command with additional environment variables
// merged into the current process environment.
func (r *RealRunner) RunWithEnv(env map[string]string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running %s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

// RunInDir executes a command in the specified working directory.
func (r *RealRunner) RunInDir(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running %s %s in %s: %w", name, strings.Join(args, " "), dir, err)
	}
	return nil
}

// RunInDirWithEnv executes a command in the specified directory with
// additional environment variables merged into the current process environment.
func (r *RealRunner) RunInDirWithEnv(dir string, env map[string]string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running %s %s in %s: %w", name, strings.Join(args, " "), dir, err)
	}
	return nil
}

// Output executes a command and returns its combined stdout as a string.
func (r *RealRunner) Output(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("running %s %s: %w", name, strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Call records a single command invocation made through FakeRunner.
type Call struct {
	Name string
	Args []string
	Env  map[string]string
	Dir  string
}

// String returns a human-readable representation of the call.
func (c Call) String() string {
	parts := []string{c.Name}
	parts = append(parts, c.Args...)
	return strings.Join(parts, " ")
}

// FakeRunner records all command invocations without executing them.
// Used in unit tests to verify the exact sequence, arguments, and
// environment of every system call.
type FakeRunner struct {
	Calls     []Call
	OutputMap map[string]string // keyed by "name arg1 arg2..." → stdout
	ErrorMap  map[string]error  // keyed by "name arg1 arg2..." → error
}

// NewFakeRunner creates a FakeRunner with initialized maps.
func NewFakeRunner() *FakeRunner {
	return &FakeRunner{
		OutputMap: make(map[string]string),
		ErrorMap:  make(map[string]error),
	}
}

func (f *FakeRunner) key(name string, args ...string) string {
	parts := []string{name}
	parts = append(parts, args...)
	return strings.Join(parts, " ")
}

// Run records the call and returns any configured error.
func (f *FakeRunner) Run(name string, args ...string) error {
	f.Calls = append(f.Calls, Call{Name: name, Args: args})
	if err, ok := f.ErrorMap[f.key(name, args...)]; ok {
		return err
	}
	return nil
}

// RunWithEnv records the call with environment and returns any configured error.
func (f *FakeRunner) RunWithEnv(env map[string]string, name string, args ...string) error {
	f.Calls = append(f.Calls, Call{Name: name, Args: args, Env: env})
	if err, ok := f.ErrorMap[f.key(name, args...)]; ok {
		return err
	}
	return nil
}

// RunInDir records the call with directory and returns any configured error.
func (f *FakeRunner) RunInDir(dir string, name string, args ...string) error {
	f.Calls = append(f.Calls, Call{Name: name, Args: args, Dir: dir})
	if err, ok := f.ErrorMap[f.key(name, args...)]; ok {
		return err
	}
	return nil
}

// RunInDirWithEnv records the call with directory and env and returns any configured error.
func (f *FakeRunner) RunInDirWithEnv(dir string, env map[string]string, name string, args ...string) error {
	f.Calls = append(f.Calls, Call{Name: name, Args: args, Dir: dir, Env: env})
	if err, ok := f.ErrorMap[f.key(name, args...)]; ok {
		return err
	}
	return nil
}

// Output records the call and returns any configured output or error.
func (f *FakeRunner) Output(name string, args ...string) (string, error) {
	f.Calls = append(f.Calls, Call{Name: name, Args: args})
	key := f.key(name, args...)
	if err, ok := f.ErrorMap[key]; ok {
		return "", err
	}
	if out, ok := f.OutputMap[key]; ok {
		return out, nil
	}
	return "", nil
}
