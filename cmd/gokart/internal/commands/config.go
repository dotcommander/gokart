package commands

import (
	"fmt"
)

type versionCommand struct{ exec *executor }

func (c *versionCommand) Run() error {
	_, err := fmt.Fprintf(c.exec.deps.Stdout, "%s version %s\n", appName, c.exec.version)
	return err
}

type configCommand struct {
	Show configShowCommand `cmd:"" help:"Print where gokart stores data."`
	exec *executor
}

type configShowCommand struct{ parent *configCommand }

func (c *configCommand) BeforeApply() error {
	c.Show.parent = c
	return nil
}

func (c *configShowCommand) Run() error {
	root, err := c.parent.exec.deps.UserConfigDir()
	if err != nil {
		root = "(unavailable: " + err.Error() + ")"
	}
	_, err = fmt.Fprintf(c.parent.exec.deps.Stdout, "Version:     %s\nConfig dir:  %s\nBinary:      %s\n", c.parent.exec.version, root, c.parent.exec.deps.BinaryPath)
	return err
}
