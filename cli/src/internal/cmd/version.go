// The version command is provided by azd-core's version.NewCommand() in
// root.go. It keeps -q/--quiet and reports build date and git commit, which
// azdext.NewVersionCommand does not, and declares its supported --output
// values through azdext.RegisterFlagOptions.
package cmd
