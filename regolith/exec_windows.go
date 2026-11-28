//go:build windows
// +build windows

package regolith

import (
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/Bedrock-OSS/go-burrito/burrito"
)

// batchFileCommand builds an exec.Cmd that runs the executable at path with
// the given arguments, passing the arguments through verbatim even when the
// executable is a batch file (.cmd or .bat).
//
// Windows runs batch files through cmd.exe, which treats characters like
// '^', '<', '>', '|' and '&' as operators even in arguments that Go quoted
// for CommandLineToArgvW. npm's launcher is a batch file (npm.cmd), and npm
// package specifications routinely contain such characters ("^1.0.0",
// ">=1.0.0 <2.0.0"), so arguments must be wrapped in double quotes and the
// whole command line handed to cmd.exe with the /s switch, which is the only
// quoting cmd.exe honours. Regular executables keep Go's default quoting.
//
// cmd.exe still expands %VAR% references inside the quotes, and a command
// line has no way to escape them. npm package names and version
// specifications can't contain '%', but the paths passed along with them
// (the --pack-destination directory, under the project or under
// %LocalAppData% with use_project_app_data_storage) can; an argument that
// names a defined environment variable between two '%' would reach the
// batch file changed, so it is rejected instead.
func batchFileCommand(path string, args ...string) (*exec.Cmd, error) {
	lowerPath := strings.ToLower(path)
	if !strings.HasSuffix(lowerPath, ".cmd") && !strings.HasSuffix(lowerPath, ".bat") {
		return exec.Command(path, args...), nil
	}
	tokens := make([]string, 0, len(args)+1)
	for _, token := range append([]string{path}, args...) {
		if strings.ContainsAny(token, "\"\r\n") {
			return nil, burrito.WrappedErrorf(
				"Unable to pass an argument to a batch file.\n"+
					"Argument: %s\n"+
					"Arguments of batch files can't contain quotes or "+
					"line breaks.", token)
		}
		if name := expandedVariable(token); name != "" {
			return nil, burrito.WrappedErrorf(
				"Unable to pass an argument to a batch file.\n"+
					"Argument: %s\n"+
					"cmd.exe would replace %%%s%% in it with the value of "+
					"the environment variable %s.", token, name, name)
		}
		tokens = append(tokens, "\""+token+"\"")
	}
	// %ComSpec% is where Windows itself looks for the command interpreter;
	// the PATH lookup is for the rare environment that doesn't set it.
	cmdExe := os.Getenv("ComSpec")
	if cmdExe == "" {
		var err error
		cmdExe, err = exec.LookPath("cmd.exe")
		if err != nil {
			return nil, burrito.WrapError(err, "Unable to find cmd.exe.")
		}
	}
	cmd := exec.Command(cmdExe)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: "cmd.exe /d /s /c \"" + strings.Join(tokens, " ") + "\"",
	}
	return cmd, nil
}

// expandedVariable returns the name of the first environment variable that
// cmd.exe would expand in token (a defined variable between two '%'), or an
// empty string when cmd.exe would pass the token through unchanged.
func expandedVariable(token string) string {
	for {
		start := strings.IndexByte(token, '%')
		if start == -1 {
			return ""
		}
		end := strings.IndexByte(token[start+1:], '%')
		if end == -1 {
			return ""
		}
		name := token[start+1 : start+1+end]
		if _, ok := os.LookupEnv(name); ok && name != "" {
			return name
		}
		// An undefined name stays literal and its closing '%' may open the
		// next reference.
		token = token[start+1+end:]
	}
}
