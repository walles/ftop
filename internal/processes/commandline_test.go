package processes

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/walles/ftop/internal/assert"
)

func TestGetTrailingAbsolutePath(t *testing.T) {
	assert.Equal(t, *getTrailingAbsolutePath("/hello"), "/hello")
	assert.Equal(t, *getTrailingAbsolutePath("/hello:/baloo"), "/baloo")

	assert.Equal(t, *getTrailingAbsolutePath("-Dx=/hello"), "/hello")
	assert.Equal(t, *getTrailingAbsolutePath("-Dx=/hello:/baloo"), "/baloo")

	assert.Equal(t, getTrailingAbsolutePath("hello"), nil)
	assert.Equal(t, getTrailingAbsolutePath("hello:/baloo"), nil)

	assert.Equal(t, *getTrailingAbsolutePath(
		"/A/IntelliJ IDEA.app/C/p/mm/lib/mm.jar:/A/IntelliJ IDEA.app/C/p/ms/lib/ms.jar:/A/IntelliJ"),
		"/A/IntelliJ")
}

func existsFromMap(outcomes map[string]existence) func(string) existence {
	return func(path string) existence {
		if outcome, ok := outcomes[path]; ok {
			return outcome
		}

		// Default: path doesn't exist, but might if we add more parts.
		// We use this for testing because it lets us write simpler test cases
		// without having to map every single parent path. The production code
		// path (the real exists() function) is correct.
		return existenceNotYet
	}
}

func mustCoalesceCount(t *testing.T, parts []string, exists func(string) existence) int {
	t.Helper()

	count, err := coalesceCount(parts, exists)
	if err != nil {
		t.Fatalf("coalesceCount() failed for %#v: %v", parts, err)
	}

	return count
}

func mustCmdlineToSlice(t *testing.T, cmdline string, exists func(string) existence) []string {
	t.Helper()

	result, err := cmdlineToSlice(cmdline, exists)
	if err != nil {
		t.Fatalf("cmdlineToSlice() failed for <%s>: %v", cmdline, err)
	}

	return result
}

func TestCoalesceCount(t *testing.T) {
	exists := existsFromMap(map[string]existence{
		"/":       existenceTrue,
		"/a b c":  existenceTrue,
		"/a b c/": existenceTrue,
	})

	assert.Equal(t, mustCoalesceCount(t, []string{"/a", "b", "c"}, exists), 3)
	assert.Equal(t, mustCoalesceCount(t, []string{"/a", "b", "c/"}, exists), 3)
	assert.Equal(t, mustCoalesceCount(t, []string{"/a", "b", "c", "d"}, exists), 3)

	assert.Equal(t,
		mustCoalesceCount(t, []string{"/a", "b", "c:/a", "b", "c"}, exists),
		5,
	)
	assert.Equal(t,
		mustCoalesceCount(t, []string{"/a", "b", "c/:/a", "b", "c/"}, exists),
		5,
	)

	assert.Equal(t,
		mustCoalesceCount(t, []string{"/a", "b", "c:/a", "b", "c", "d"}, exists),
		5,
	)
	assert.Equal(t,
		mustCoalesceCount(t, []string{"/a", "b", "c/:/a", "b", "c/", "d/"}, exists),
		5,
	)
}

func TestIsFileNameTooLong(t *testing.T) {
	tooLong := strings.Repeat("a", 1234)
	_, err := os.Stat(tooLong)
	assert.Equal(t, true, isFileNameTooLong(err))
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	existingFile := filepath.Join(dir, "existing")
	err := os.WriteFile(existingFile, []byte(""), 0o644)
	assert.Equal(t, err, nil)

	assert.Equal(t, exists(existingFile), existenceTrue)
	assert.Equal(t, exists(filepath.Join(dir, "existi")), existenceNotYet)

	tooLong := strings.Repeat("a", 1234)
	assert.Equal(t, exists(filepath.Join(dir, tooLong)), existenceFalse)
}

func TestToSliceSpaced1(t *testing.T) {
	exists := existsFromMap(map[string]existence{
		"/Applications":                            existenceTrue,
		"/Applications/IntelliJ IDEA.app":          existenceTrue,
		"/Applications/IntelliJ IDEA.app/Contents": existenceTrue,
	})

	result := mustCmdlineToSlice(t,
		"java -Dhello=/Applications/IntelliJ IDEA.app/Contents",
		exists,
	)

	assert.SlicesEqual(t, result, []string{
		"java",
		"-Dhello=/Applications/IntelliJ IDEA.app/Contents",
	})
}

func TestToSliceSpaced2(t *testing.T) {
	exists := existsFromMap(map[string]existence{
		"/Applications": existenceTrue,
		"/Applications/IntelliJ IDEA.app/Contents/Info.plist":                                 existenceTrue,
		"/Applications/IntelliJ IDEA.app/Contents/plugins/maven-model/lib/maven-model.jar":    existenceTrue,
		"/Applications/IntelliJ IDEA.app/Contents/plugins/maven-server/lib/maven-server.jar":  existenceTrue,
		"/Applications/IntelliJ IDEA.app/Contents/plugins/maven/lib/maven3-server-common.jar": existenceTrue,
	})

	result := mustCmdlineToSlice(t, strings.Join([]string{
		"java",
		"-Dhello=/Applications/IntelliJ IDEA.app/Contents/Info.plist",
		"-classpath",
		"/Applications/IntelliJ",
		"IDEA.app/Contents/plugins/maven-model/lib/maven-model.jar:/Applications/IntelliJ",
		"IDEA.app/Contents/plugins/maven-server/lib/maven-server.jar:/Applications/IntelliJ",
		"IDEA.app/Contents/plugins/maven/lib/maven3-server-common.jar",
		"MainClass",
	}, " "), exists)

	assert.SlicesEqual(t, result, []string{
		"java",
		"-Dhello=/Applications/IntelliJ IDEA.app/Contents/Info.plist",
		"-classpath",
		"/Applications/IntelliJ IDEA.app/Contents/plugins/maven-model/lib/maven-model.jar:/Applications/IntelliJ IDEA.app/Contents/plugins/maven-server/lib/maven-server.jar:/Applications/IntelliJ IDEA.app/Contents/plugins/maven/lib/maven3-server-common.jar",
		"MainClass",
	})
}

func TestToSliceSpaced3(t *testing.T) {
	exists := existsFromMap(map[string]existence{
		"/Applications": existenceTrue,
		"/Applications/IntelliJ IDEA CE.app/Contents/Info.plist":                                 existenceTrue,
		"/Applications/IntelliJ IDEA CE.app/Contents/plugins/maven-model/lib/maven-model.jar":    existenceTrue,
		"/Applications/IntelliJ IDEA CE.app/Contents/plugins/maven-server/lib/maven-server.jar":  existenceTrue,
		"/Applications/IntelliJ IDEA CE.app/Contents/plugins/maven/lib/maven3-server-common.jar": existenceTrue,
	})

	result := mustCmdlineToSlice(t, strings.Join([]string{
		"java",
		"-Dhello=/Applications/IntelliJ IDEA CE.app/Contents/Info.plist",
		"-classpath",
		"/Applications/IntelliJ",
		"IDEA",
		"CE.app/Contents/plugins/maven-model/lib/maven-model.jar:/Applications/IntelliJ",
		"IDEA",
		"CE.app/Contents/plugins/maven-server/lib/maven-server.jar:/Applications/IntelliJ",
		"IDEA",
		"CE.app/Contents/plugins/maven/lib/maven3-server-common.jar",
		"MainClass",
	}, " "), exists)

	assert.SlicesEqual(t, result, []string{
		"java",
		"-Dhello=/Applications/IntelliJ IDEA CE.app/Contents/Info.plist",
		"-classpath",
		"/Applications/IntelliJ IDEA CE.app/Contents/plugins/maven-model/lib/maven-model.jar:/Applications/IntelliJ IDEA CE.app/Contents/plugins/maven-server/lib/maven-server.jar:/Applications/IntelliJ IDEA CE.app/Contents/plugins/maven/lib/maven3-server-common.jar",
		"MainClass",
	})
}

func TestToSliceMsEdge(t *testing.T) {
	complete := strings.Join([]string{
		"/Applications",
		"Microsoft Edge.app",
		"Contents",
		"Frameworks",
		"Microsoft Edge Framework.framework",
		"Versions",
		"122.0.2365.63",
		"Helpers",
		"Microsoft Edge Helper (GPU).app",
		"Contents",
		"MacOS",
		"Microsoft Edge Helper (GPU)",
	}, "/")

	outcomes := make(map[string]existence)
	partial := complete
	for {
		outcomes[partial] = existenceTrue
		partial = path.Dir(partial)
		if partial == "/" {
			break
		}
	}

	exists := existsFromMap(outcomes)

	result := mustCmdlineToSlice(t, complete+" --type=gpu-process", exists)

	assert.SlicesEqual(t, result, []string{complete, "--type=gpu-process"})
}

func TestDotnetCommandline(t *testing.T) {
	// Unclear whether "fable" is a builtin or a separate tool, go with "dotnet fable".
	assert.Equal(t,
		cmdlineToCommand("libexec/dotnet fable core/Core.Test.fsproj", nil),
		"dotnet fable",
	)

	// The DLL has a path, so it can't be a builtin. Go with just "fable.dll"
	assert.Equal(t,
		cmdlineToCommand("libexec/dotnet any/fable.dll core/Core.Test.fsproj", nil),
		"fable.dll",
	)
}

func TestNodeMaxOldSpace(t *testing.T) {
	assert.Equal(t,
		cmdlineToCommand("node --max_old_space_size=4096 scripts/start.js", nil),
		"start.js",
	)
}

func TestGetBashBrewShCommandline(t *testing.T) {
	assert.Equal(t,
		cmdlineToCommand("/bin/bash -p /usr/local/Homebrew/Library/Homebrew/brew.sh upgrade", nil),
		"brew.sh upgrade",
	)
}

func TestGetCommandInterpreters(t *testing.T) {
	// ruby
	assert.Equal(t, cmdlineToCommand("ruby", nil), "ruby")
	assert.Equal(t, cmdlineToCommand("ruby /some/path/apa.rb", nil), "apa.rb")
	assert.Equal(t, cmdlineToCommand("ruby -option /some/path/apa.rb", nil), "ruby")

	// sh
	assert.Equal(t, cmdlineToCommand("sh", nil), "sh")
	assert.Equal(t, cmdlineToCommand("sh /some/path/apa.sh", nil), "apa.sh")
	assert.Equal(t, cmdlineToCommand("sh -option /some/path/apa.sh", nil), "sh")

	// bash
	assert.Equal(t, cmdlineToCommand("bash", nil), "bash")
	assert.Equal(t, cmdlineToCommand("bash /some/path/apa.sh", nil), "apa.sh")
	assert.Equal(t, cmdlineToCommand("bash -option /some/path/apa.sh", nil), "bash")

	// perl
	assert.Equal(t, cmdlineToCommand("perl", nil), "perl")
	assert.Equal(t, cmdlineToCommand("perl /some/path/apa.pl", nil), "apa.pl")
	assert.Equal(t, cmdlineToCommand("perl -option /some/path/apa.pl", nil), "perl")
}

func TestGetGoCommandline(t *testing.T) {
	assert.Equal(t, cmdlineToCommand("go build ./...", nil), "go build")
	assert.Equal(t, cmdlineToCommand("go --version", nil), "go")
	assert.Equal(t, cmdlineToCommand("/usr/local/bin/go", nil), "go")
}

func TestGetGitCommandline(t *testing.T) {
	assert.Equal(t, cmdlineToCommand("git clone git@github.com:walles/riff", nil), "git clone")
	assert.Equal(t, cmdlineToCommand("git --version", nil), "git")
	assert.Equal(t, cmdlineToCommand("/usr/local/bin/git", nil), "git")
	assert.Equal(t, cmdlineToCommand("git -c core.quotepath=false reflog --max-count 50", nil), "git reflog")
	assert.Equal(t, cmdlineToCommand("git -c core.quotepath=false", nil), "git")
	assert.Equal(t, cmdlineToCommand("git -c", nil), "git")
	assert.Equal(t, cmdlineToCommand("git -C /tmp/hello show", nil), "git show")
}

func TestGetTerraformCommandline(t *testing.T) {
	assert.Equal(t, cmdlineToCommand("terraform -chdir=dev apply -target=abc123", nil), "terraform apply")
}

func TestGetTerraformProviderCommandline(t *testing.T) {
	// Source: https://github.com/walles/px/issues/105
	assert.Equal(t,
		cmdlineToCommand(".terraform/providers/registry.terraform.io/heroku/heroku/4.8.0/darwin_amd64/terraform-provider-heroku_v4.8.0", nil),
		"terraform-provider-heroku_v4.8.0",
	)
}

func TestGetCommandPython(t *testing.T) {
	// Basics
	assert.Equal(t, cmdlineToCommand("python", nil), "python")
	assert.Equal(t, cmdlineToCommand("/apa/Python", nil), "Python")
	assert.Equal(t, cmdlineToCommand("python --help", nil), "python")

	// Running scripts / modules
	assert.Equal(t, cmdlineToCommand("python apa.py", nil), "apa.py")
	assert.Equal(t, cmdlineToCommand("python /usr/bin/apa.py", nil), "apa.py")
	assert.Equal(t, cmdlineToCommand("python2.7 /usr/bin/apa.py", nil), "apa.py")
	assert.Equal(t, cmdlineToCommand("python /usr/bin/hej", nil), "hej")
	assert.Equal(t, cmdlineToCommand("python /usr/bin/hej gris --flaska", nil), "hej")
	assert.Equal(t, cmdlineToCommand("python -c cmd", nil), "python")
	assert.Equal(t, cmdlineToCommand("python -m mod", nil), "mod")
	assert.Equal(t, cmdlineToCommand("python -m mod --hej gris --frukt", nil), "mod")
	assert.Equal(t, cmdlineToCommand("Python -", nil), "Python")

	// Ignoring switches
	assert.Equal(t, cmdlineToCommand("python -E apa.py", nil), "apa.py")
	assert.Equal(t, cmdlineToCommand("python3 -E", nil), "python3")
	assert.Equal(t, cmdlineToCommand("python -u -t -m mod", nil), "mod")

	// -W switches unsupported for now
	assert.Equal(t, cmdlineToCommand("python -W warning:spec apa.py", nil), "python")

	// Invalid command lines
	assert.Equal(t, cmdlineToCommand("python -W", nil), "python")
	assert.Equal(t, cmdlineToCommand("python -c", nil), "python")
	assert.Equal(t, cmdlineToCommand("python -m", nil), "python")
	assert.Equal(t, cmdlineToCommand("python -m   ", nil), "python")
	assert.Equal(t, cmdlineToCommand("python -m -u", nil), "python")
	assert.Equal(t, cmdlineToCommand("python    ", nil), "python")
}

func TestGetCommandAws(t *testing.T) {
	// Python wrapper around aws
	assert.Equal(t, cmdlineToCommand("Python /usr/local/bin/aws", nil), "aws")
	assert.Equal(t, cmdlineToCommand("python aws s3", nil), "aws s3")
	assert.Equal(t, cmdlineToCommand("python3 aws s3 help", nil), "aws s3 help")
	assert.Equal(t, cmdlineToCommand("/wherever/python3 aws s3 help flaska", nil), "aws s3 help")

	assert.Equal(t, cmdlineToCommand("python aws s3 sync help", nil), "aws s3 sync help")
	assert.Equal(t, cmdlineToCommand("python aws s3 sync nothelp", nil), "aws s3 sync")
	assert.Equal(t, cmdlineToCommand("python aws s3 --unknown sync", nil), "aws s3")

	// Ignore profile and region; stop at switches and paths
	cmd := strings.Join([]string{
		"python3",
		"/usr/local/bin/aws",
		"--profile=system-admin-prod",
		"--region=eu-west-1",
		"s3",
		"sync",
		"--only-show-errors",
		"s3://xxxxxx",
		"./xxxxxx",
	}, " ")
	assert.Equal(t, cmdlineToCommand(cmd, nil), "aws s3 sync")
}

func TestGetCommandLoginShell(t *testing.T) {
	assert.Equal(t, cmdlineToCommand("-fish", nil), "fish")
}

func TestGetCommandSudo(t *testing.T) {
	assert.Equal(t, cmdlineToCommand("sudo", nil), "sudo")
	assert.Equal(t, cmdlineToCommand("sudo python /usr/bin/hej gris --flaska", nil), "sudo hej")

	// With flags we give up and keep just "sudo"
	assert.Equal(t, cmdlineToCommand("sudo -B python /usr/bin/hej", nil), "sudo")
}

func TestGetCommandSudoWithSpaceInCommandName(t *testing.T) {
	dir := t.TempDir()
	spacedPath := filepath.Join(dir, "i contain spaces")
	err := os.WriteFile(spacedPath, []byte(""), 0o644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Verify splitting of the spaced file name
	assert.Equal(t, cmdlineToCommand("sudo "+spacedPath, nil), "sudo i contain spaces")

	// Verify splitting with more parameters on the line
	assert.Equal(t, cmdlineToCommand("sudo "+spacedPath+" parameter", nil), "sudo i contain spaces")
}

func TestGetCommandSudoWithSpaceInPath(t *testing.T) {
	dir := t.TempDir()
	spacedDir := filepath.Join(dir, "i contain spaces")
	if err := os.MkdirAll(spacedDir, 0o755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	spacedPath := filepath.Join(spacedDir, "runme")
	if err := os.WriteFile(spacedPath, []byte(""), 0o755); err != nil {
		t.Fatalf("failed to create runme file: %v", err)
	}

	// Verify splitting of the spaced file name
	assert.Equal(t, cmdlineToCommand("sudo "+spacedPath, nil), "sudo runme")

	// Verify splitting with more parameters on the line
	assert.Equal(t, cmdlineToCommand("sudo "+spacedPath+" parameter", nil), "sudo runme")
}

// Ref: https://github.com/walles/ftop/issues/5
func TestIgnoreShellCdAndAnd(t *testing.T) {
	assert.Equal(t, cmdlineToCommand("/bin/sh -c cd ~/src/moor && moor twin/screen.go", nil), "moor")
}

func TestIgnoreLeadingShellC(t *testing.T) {
	assert.Equal(t, cmdlineToCommand("/bin/sh -c which minikube", nil), "which minikube")
}

func TestGetCommandRubySwitches(t *testing.T) {
	// ruby with warning level switch and brew.rb subcommand
	assert.Equal(t,
		cmdlineToCommand("/usr/bin/ruby -W0 /usr/local/bin/brew.rb install rust", nil),
		"brew.rb install",
	)

	// Double-dash to end options, should pick the first script after it
	assert.Equal(t,
		cmdlineToCommand("/usr/bin/ruby -W1 -- /apa/build.rb /bepa/cmake.rb", nil),
		"build.rb",
	)
	assert.Equal(t,
		cmdlineToCommand("/usr/bin/ruby -W1 -- -hello.rb /bepa/cmake.rb", nil),
		"-hello.rb",
	)

	// Encoding switch should be ignored
	assert.Equal(t,
		cmdlineToCommand("/usr/bin/ruby -Eascii-8bit:ascii-8bit /usr/sbin/google-fluentd", nil),
		"google-fluentd",
	)

	// -I switch and its argument should be ignored...
	assert.Equal(t,
		cmdlineToCommand("/usr/bin/ruby -I /some/include /usr/local/bin/brew.rb update", nil),
		"brew.rb update",
	)

	// ... but if no includes follow, give up
	assert.Equal(t,
		cmdlineToCommand("/usr/bin/ruby -I", nil),
		"ruby",
	)
}

func TestGetCommandPerl(t *testing.T) {
	// Variants should all resolve to the script name
	assert.Equal(t,
		cmdlineToCommand(strings.Join([]string{
			"/usr/bin/perl5.18",
			"/usr/local/Cellar/cloc/1.90/libexec/bin/cloc",
			"build-system",
			"build_number_offset",
			"buildbox",
			"Random.txt",
			"README.md",
			"submodules",
			"Telegram",
			"third-party",
			"tools",
			"versions.json",
			"WORKSPACE",
		}, " "), nil),
		"cloc",
	)

	assert.Equal(t,
		cmdlineToCommand("/usr/bin/perl /usr/local/Cellar/cloc/1.90/libexec/bin/cloc", nil),
		"cloc",
	)
	assert.Equal(t,
		cmdlineToCommand("perl /usr/local/Cellar/cloc/1.90/libexec/bin/cloc", nil),
		"cloc",
	)
	assert.Equal(t,
		cmdlineToCommand("/usr/bin/perl5 /usr/local/Cellar/cloc/1.90/libexec/bin/cloc", nil),
		"cloc",
	)
	assert.Equal(t,
		cmdlineToCommand("/usr/bin/perl5.30 /usr/local/Cellar/cloc/1.90/libexec/bin/cloc", nil),
		"cloc",
	)

	// Give up on command line switches
	assert.Equal(t,
		cmdlineToCommand("/usr/bin/perl -S cloc", nil),
		"perl",
	)
}

func TestMacosApp(t *testing.T) {
	// Dock external extra XPC service
	dock := strings.Join([]string{
		"/System",
		"Library",
		"CoreServices",
		"Dock.app",
		"Contents",
		"XPCServices",
		"com.apple.dock.external.extra.xpc",
		"Contents",
		"MacOS",
		"com.apple.dock.external.extra",
	}, "/")
	assert.Equal(t, cmdlineToCommand(dock, nil), "Dock/extra")

	// Firefox plugin-container nested .app
	firefox := strings.Join([]string{
		"/Applications",
		"Firefox.app",
		"Contents",
		"MacOS",
		"plugin-container.app",
		"Contents",
		"MacOS",
		"plugin-container",
	}, "/")
	assert.Equal(t, cmdlineToCommand(firefox, nil), "Firefox/plugin-container")

	// CodeHelper(Renderer)
	codeHelper := strings.Join([]string{
		"/Applications",
		"VisualStudioCode.app",
		"Contents",
		"Frameworks",
		"CodeHelper(Renderer).app",
		"Contents",
		"MacOS",
		"CodeHelper(Renderer)",
	}, "/")
	assert.Equal(t, cmdlineToCommand(codeHelper, nil), "CodeHelper(Renderer)")

	// iTerm without prefix (human-friendly command)
	assert.Equal(t, cmdlineToCommand("/Applications/iTerm.app/Contents/MacOS/iTerm2", nil), "iTerm2")

	// IDS.framework without duplicating name
	ids := strings.Join([]string{
		"/System",
		"Library",
		"PrivateFrameworks",
		"IDS.framework",
		"identityservicesd.app",
		"Contents",
		"MacOS",
		"identityservicesd",
	}, "/")
	assert.Equal(t, cmdlineToCommand(ids, nil), "IDS/identityservicesd")

	intelligencePlatform := strings.Join([]string{
		"/System",
		"/Library",
		"/PrivateFrameworks",
		"/IntelligencePlatformCore.framework",
		"/Versions",
		"/A",
		"/intelligenceplatformd",
	}, "/")
	assert.Equal(t, cmdlineToCommand(intelligencePlatform, nil), "IntelligencePlatformCore (Daemon)")
}

func TestCoalesceAppCommand(t *testing.T) {
	// Main example from the function comment
	assert.Equal(t,
		coalesceAppCommand("GenerativeExperiencesRuntime/generativeexperiencesd"),
		"GenerativeExperiencesRuntime (Daemon)",
	)

	// Real-world example from macOS
	assert.Equal(t,
		coalesceAppCommand("IntelligencePlatformCore/intelligenceplatformd"),
		"IntelligencePlatformCore (Daemon)",
	)

	// No slash - return as-is
	assert.Equal(t,
		coalesceAppCommand("intelligenceplatformd"),
		"intelligenceplatformd",
	)

	// Multiple slashes - return as-is
	assert.Equal(t,
		coalesceAppCommand("IntelligencePlatformCore/Versions/intelligenceplatformd"),
		"IntelligencePlatformCore/Versions/intelligenceplatformd",
	)

	// Second part without 'd' is not a prefix of first - return as-is
	assert.Equal(t,
		coalesceAppCommand("MyApp/otherthingd"),
		"MyApp/otherthingd",
	)

	// Short second part that's a prefix
	assert.Equal(t,
		coalesceAppCommand("Application/appd"),
		"Application (Daemon)",
	)

	assert.Equal(t,
		coalesceAppCommand("Software Update/softwareupdated"),
		"Software Update (Daemon)",
	)

	assert.Equal(t,
		coalesceAppCommand("Software-Update/softwareupdated"),
		"Software Update (Daemon)",
	)
}

func TestGetCommandElectronMacos(t *testing.T) {
	assert.Equal(t,
		cmdlineToCommand(strings.Join([]string{
			"/Applications/VisualStudioCode.app/Contents/MacOS/Electron",
			"--ms-enable-electron-run-as-node",
			"/Users/johan/.vscode/extensions/ms-python.vscode-pylance-2021.12.2/dist/server.bundle.js",
			"--cancellationReceive=file:d6fe53594ec46a8bb986ad058c985f56d309e7bf19",
			"--node-ipc",
			"--clientProcessId=42516",
		}, " "), nil),
		"VisualStudioCode",
	)
}

func TestGetCommandFlutter(t *testing.T) {
	// Subcommand without path/dots -> treat as subcommand
	assert.Equal(t,
		cmdlineToCommand(strings.Join([]string{
			"/usr/local/Cellar/dart/2.15.1/libexec/bin/dart",
			"devtools",
		}, " "), nil),
		"dart devtools",
	)

	// Dotted candidate -> treat as file
	assert.Equal(t,
		cmdlineToCommand(strings.Join([]string{
			"/usr/local/Cellar/dart/2.15.1/libexec/bin/dart",
			"devtools.snap",
		}, " "), nil),
		"devtools.snap",
	)

	// Path candidate -> basename
	assert.Equal(t,
		cmdlineToCommand(strings.Join([]string{
			"/usr/local/Cellar/dart/2.15.1/libexec/bin/dart",
			"/usr/local/bin/devtools",
		}, " "), nil),
		"devtools",
	)
}

func TestGetCommandGuile(t *testing.T) {
	// Plain guile
	assert.Equal(t, cmdlineToCommand("guile", nil), "guile")

	// -l returns its arg as the script
	assert.Equal(t,
		cmdlineToCommand("guile -l myscript.scm", nil),
		"myscript.scm",
	)

	// Ignore common switches
	assert.Equal(t,
		cmdlineToCommand("guile -q --r6rs myscript.scm", nil),
		"myscript.scm",
	)

	// Ignore argful switches
	assert.Equal(t,
		cmdlineToCommand("guile -L /usr/local/include -C /tmp -x scheme myscript.scm", nil),
		"myscript.scm",
	)

	// --listen with equals form
	assert.Equal(t,
		cmdlineToCommand("guile --listen=localhost:37146 myscript.scm", nil),
		"myscript.scm",
	)

	// Unknown switch after guile -> give up
	assert.Equal(t,
		cmdlineToCommand("guile --unknown myscript.scm", nil),
		"guile",
	)
}

func TestGetCommandResque(t *testing.T) {
	assert.Equal(t, cmdlineToCommand("resque-1.20.0: a b c", nil), "resque-1.20.0:")
	assert.Equal(t, cmdlineToCommand("resqued-0.7.12 x y z", nil), "resqued-0.7.12")
}

func TestGetHomebrewCommandline(t *testing.T) {
	assert.Equal(t,
		cmdlineToCommand(strings.Join([]string{
			"/usr/local/Homebrew/Library/Homebrew/vendor/portable-ruby/current/bin/ruby",
			"-W0",
			"--disable=gems,did_you_mean,rubyopt",
			"/usr/local/Homebrew/Library/Homebrew/brew.rb",
			"upgrade",
		}, " "), nil),
		"brew.rb upgrade",
	)
}

func TestGetCommandUnicode(t *testing.T) {
	// Emoji-only command should be preserved
	assert.Equal(t, cmdlineToCommand("😀", nil), "😀")

	// Simple unicode executable name
	assert.Equal(t, cmdlineToCommand("/usr/local/bin/äpple", nil), "äpple")

	// Python running a unicode script path -> basename
	assert.Equal(t, cmdlineToCommand("python /usr/bin/hällo.py", nil), "hällo.py")

	// Ruby running a unicode script path -> basename
	assert.Equal(t, cmdlineToCommand("ruby /some/path/тест.rb", nil), "тест.rb")

	// Shell running a unicode script path -> basename
	assert.Equal(t, cmdlineToCommand("bash /some/path/ユニコード.sh", nil), "ユニコード.sh")
}

func TestGetCommandJava(t *testing.T) {
	// Basics
	assert.Equal(t, cmdlineToCommand("java", nil), "java")
	assert.Equal(t, cmdlineToCommand("java -version", nil), "java")
	assert.Equal(t, cmdlineToCommand("java -help", nil), "java")

	// Class and jar
	assert.Equal(t, cmdlineToCommand("java SomeClass", nil), "SomeClass")
	assert.Equal(t, cmdlineToCommand("java x.y.SomeClass", nil), "SomeClass")
	assert.Equal(t, cmdlineToCommand("java -jar flaska.jar", nil), "flaska.jar")
	assert.Equal(t, cmdlineToCommand("java -jar /a/b/flaska.jar", nil), "flaska.jar")

	// Special handling of Main
	assert.Equal(t, cmdlineToCommand("java a.b.c.Main", nil), "c.Main")

	// Ignore certain options
	assert.Equal(t, cmdlineToCommand("java -server SomeClass", nil), "SomeClass")
	assert.Equal(t, cmdlineToCommand("java -Xwhatever SomeClass", nil), "SomeClass")
	assert.Equal(t, cmdlineToCommand("java -Dwhatever SomeClass", nil), "SomeClass")
	assert.Equal(t, cmdlineToCommand("java -eahej SomeClass", nil), "SomeClass")
	assert.Equal(t, cmdlineToCommand("java -dahej SomeClass", nil), "SomeClass")
	assert.Equal(t, cmdlineToCommand("java -cp /a/b/c SomeClass", nil), "SomeClass")
	assert.Equal(t, cmdlineToCommand("java -classpath /a/b/c SomeClass", nil), "SomeClass")
	assert.Equal(t, cmdlineToCommand("java --enable-native-access=ALL-UNNAMED SomeClass", nil), "SomeClass")

	// Invalid command lines
	assert.Equal(t, cmdlineToCommand("java -cp /a/b/c", nil), "java")
	assert.Equal(t, cmdlineToCommand("java  ", nil), "java")
	assert.Equal(t, cmdlineToCommand("java -jar", nil), "java")
	assert.Equal(t, cmdlineToCommand("java -jar    ", nil), "java")
}

func TestGetCommandJavaEquinox(t *testing.T) {
	commandline := strings.Join([]string{
		"/Library/Java/JavaVirtualMachines/openjdk-11.jdk/Contents/Home/bin/java",
		"--add-modules=ALL-SYSTEM",
		"--add-opens",
		"java.base/java.util=ALL-UNNAMED",
		"--add-opens",
		"java.base/java.lang=ALL-UNNAMED",
		"-Declipse.application=org.eclipse.jdt.ls.core.id1",
		"-Dosgi.bundles.defaultStartLevel=4",
		"-Declipse.product=org.eclipse.jdt.ls.core.product",
		"-Dfile.encoding=utf8",
		"-XX:+UseParallelGC",
		"-XX:GCTimeRatio=4",
		"-XX:AdaptiveSizePolicyWeight=90",
		"-Dsun.zip.disableMemoryMapping=true",
		"-Xmx1G",
		"-Xms100m",
		"-noverify",
		"-jar",
		"/Users/walles/.vscode/extensions/redhat.java-0.68.0/server/plugins/org.eclipse.equinox.launcher_1.5.800.v20200727-1323.jar",
		"-configuration",
		"/Users/walles/Library/Application Support/Code/User/globalStorage/redhat.java/0.68.0/config_mac",
		"-data",
		"/Users/walles/Library/Application Support/Code/User/workspaceStorage/b8c3a38f62ce0fc92ce4edfb836480db/redhat.java/jdt_ws",
	}, " ")

	assert.Equal(t, cmdlineToCommand(commandline, nil), "org.eclipse.equinox.launcher_1.5.800.v20200727-1323.jar")
}

func TestGetCommandJavaGradled(t *testing.T) {
	commandline := strings.Join([]string{
		"/Library/Java/JavaVirtualMachines/jdk1.8.0_60.jdk/Contents/Home/bin/java",
		"-XX:MaxPermSize=256m",
		"-XX:+HeapDumpOnOutOfMemoryError",
		"-Xmx1024m",
		"-Dfile.encoding=UTF-8",
		"-Duser.country=SE",
		"-Duser.language=sv",
		"-Duser.variant",
		"-cp",
		"/Users/johan/.gradle/wrapper/dists/gradle-2.8-all/gradle-2.8/lib/gradle-launcher-2.8.jar",
		"org.gradle.launcher.daemon.bootstrap.GradleDaemon",
		"2.8",
	}, " ")

	assert.Equal(t, cmdlineToCommand(commandline, nil), "GradleDaemon")
}

func TestGetCommandJavaGradleWorkerMain(t *testing.T) {
	commandline := strings.Join([]string{
		"/some/path/bin/java",
		"-Djava.awt.headless=true",
		"-Djava.security.manager=worker.org.gradle.process.internal.worker.child.BootstrapSecurityManager",
		"-Dorg.gradle.native=false",
		"-Drobolectric.accessibility.enablechecks=true",
		"-Drobolectric.logging=stderr",
		"-Drobolectric.logging.enabled=true",
		"-agentlib:jdwp=transport=dt_socket,server=y,address=,suspend=n",
		"-noverify",
		"-javaagent:gen_build/.../jacocoagent.jar=destfile=gen_build/jacoco/testDebugUnitTest.exec,append=true,dumponexit=true,output=file,jmx=false",
		"-Xmx2400m",
		"-Dfile.encoding=UTF-8",
		"-Duser.country=SE",
		"-Duser.language=sv",
		"-Duser.variant",
		"-ea",
		"-cp",
		"/Users/walles/.gradle/caches/4.2.1/workerMain/gradle-worker.jar",
		"worker.org.gradle.process.internal.worker.GradleWorkerMain",
		"'Gradle Test Executor 16'",
	}, " ")

	assert.Equal(t, cmdlineToCommand(commandline, nil), "GradleWorkerMain")
}

func TestGetCommandJavaLogstash(t *testing.T) {
	commandline := strings.Join([]string{
		"/usr/bin/java",
		"-XX:+UseParNewGC",
		"-XX:+UseConcMarkSweepGC",
		"-Djava.awt.headless=true",
		"-XX:CMSInitiatingOccupancyFraction=75",
		"-XX:+UseCMSInitiatingOccupancyOnly",
		"-Djava.io.tmpdir=/var/lib/logstash",
		"-Xmx128m",
		"-Xss2048k",
		"-Djffi.boot.library.path=/opt/logstash/vendor/jruby/lib/jni",
		"-XX:+UseParNewGC",
		"-XX:+UseConcMarkSweepGC",
		"-Djava.awt.headless=true",
		"-XX:CMSInitiatingOccupancyFraction=75",
		"-XX:+UseCMSInitiatingOccupancyOnly",
		"-Djava.io.tmpdir=/var/lib/logstash",
		"-Xbootclasspath/a:/opt/logstash/vendor/jruby/lib/jruby.jar",
		"-classpath",
		":",
		"-Djruby.home=/opt/logstash/vendor/jruby",
		"-Djruby.lib=/opt/logstash/vendor/jruby/lib",
		"-Djruby.script=jruby",
		"-Djruby.shell=/bin/sh",
		"org.jruby.Main",
		"--1.9",
		"/opt/logstash/lib/bootstrap/environment.rb",
		"logstash/runner.rb",
		"agent",
		"-f",
		"/etc/logstash/conf.d",
		"-l",
		"/var/log/logstash/logstash.log",
	}, " ")

	assert.Equal(t, cmdlineToCommand(commandline, nil), "jruby.Main")
}

func TestGetCommandJavaTeamCity(t *testing.T) {
	commandline := strings.Join([]string{
		"/usr/lib/jvm/jdk-8-oracle-x64/jre/bin/java",
		"-Djava.util.logging.config.file=/teamcity/conf/logging.properties",
		"-Djava.util.logging.manager=org.apache.juli.ClassLoaderLogManager",
		"-Dsun.net.inetaddr.ttl=60",
		"-server",
		"-Xms31g",
		"-Xmx31g",
		"-Dteamcity.configuration.path=../conf/teamcity-startup.properties",
		"-Dlog4j.configuration=file:/teamcity/bin/../conf/teamcity-server-log4j.xml",
		"-Dteamcity_logs=../logs/",
		"-Djsse.enableSNIExtension=false",
		"-Djava.awt.headless=true",
		"-Djava.endorsed.dirs=/teamcity/endorsed",
		"-classpath",
		"/teamcity/bin/bootstrap.jar:/teamcity/bin/tomcat-juli.jar",
		"-Dcatalina.base=/teamcity",
		"-Dcatalina.home=/teamcity",
		"-Djava.io.tmpdir=/teamcity/temp",
		"org.apache.catalina.startup.Bootstrap",
		"start",
	}, " ")

	assert.Equal(t, cmdlineToCommand(commandline, nil), "Bootstrap")
}

func TestGetCommandLineJavaIssue139(t *testing.T) {
	commandline := strings.Join([]string{
		"/opt/homebrew/Cellar/openjdk@21/21.0.8/libexec/openjdk.jdk/Contents/Home/bin/java",
		"--add-exports=jdk.compiler/com.sun.tools.javac.api=ALL-UNNAMED",
		"--add-exports=jdk.compiler/com.sun.tools.javac.file=ALL-UNNAMED",
		"--add-exports=jdk.compiler/com.sun.tools.javac.main=ALL-UNNAMED",
		"--add-exports=jdk.compiler/com.sun.tools.javac.model=ALL-UNNAMED",
		"--add-exports=jdk.compiler/com.sun.tools.javac.parser=ALL-UNNAMED",
		"--add-exports=jdk.compiler/com.sun.tools.javac.processing=ALL-UNNAMED",
		"--add-exports=jdk.compiler/com.sun.tools.javac.tree=ALL-UNNAMED",
		"--add-exports=jdk.compiler/com.sun.tools.javac.util=ALL-UNNAMED",
		"@/Users/johan/.gradle/.tmp/gradle-worker-classpath12697647132064750531txt",
		"-Xmx2g",
		"-Dfile.encoding=UTF-8",
		"-Duser.country=SE",
		"-Duser.language=sv",
		"-Duser.variant",
		"worker.org.gradle.process.internal.worker.GradleWorkerMain",
		"'Gradle Worker Daemon 3'",
	}, " ")

	assert.Equal(t, cmdlineToCommand(commandline, nil), "GradleWorkerMain")
}

func TestGetCommandLineJdkJavaOptions(t *testing.T) {
	commandline := strings.Join([]string{
		"/opt/homebrew/Cellar/openjdk/24.0.1/libexec/openjdk.jdk/Contents/Home/bin/java",
		"-classpath",
		"/opt/homebrew/Cellar/mvnd/2.0.0-rc-3/libexec/mvn/boot/plexus-classworlds-2.8.0.jar",
		"-javaagent:/opt/homebrew/Cellar/mvnd/2.0.0-rc-3/libexec/mvn/lib/mvnd/mvnd-agent-2.0.0-rc-3.jar",
		"-Dmvnd.home=/opt/homebrew/Cellar/mvnd/2.0.0-rc-3/libexec",
		"-Dmaven.home=/opt/homebrew/Cellar/mvnd/2.0.0-rc-3/libexec/mvn",
		"-Dmaven.conf=/opt/homebrew/Cellar/mvnd/2.0.0-rc-3/libexec/mvn/conf",
		"-Dclassworlds.conf=/opt/homebrew/Cellar/mvnd/2.0.0-rc-3/libexec/bin/mvnd-daemon.conf",
		"-Dmaven.logger.logFile=/Users/johan/.m2/mvnd/registry/2.0.0-rc-3/daemon-802431a9.log",
		"-Dmvnd.java.home=/opt/homebrew/Cellar/openjdk/24.0.1/libexec/openjdk.jdk/Contents/Home",
		"-Dmvnd.id=802431a9",
		"-Dmvnd.daemonStorage=/Users/johan/.m2/mvnd/registry/2.0.0-rc-3",
		"-Dmvnd.registry=/Users/johan/.m2/mvnd/registry/2.0.0-rc-3/registry.bin",
		"-Dmvnd.socketFamily=inet",
		"-Djdk.java.options=--add-opens",
		"java.base/java.io=ALL-UNNAMED",
		"--add-opens",
		"java.base/java.lang=ALL-UNNAMED",
		"--add-opens",
		"java.base/java.util=ALL-UNNAMED",
		"--add-opens",
		"java.base/jdk.internal.misc=ALL-UNNAMED",
		"--add-opens",
		"java.base/sun.net.www.protocol.jar=ALL-UNNAMED",
		"--add-opens",
		"java.base/sun.nio.fs=ALL-UNNAMED",
		"-Dmvnd.noDaemon=false",
		"-Dmvnd.debug=false",
		"-Dmvnd.debug.address=8000",
		"-Dmvnd.idleTimeout=3h",
		"-Dmvnd.keepAlive=100ms",
		"-Dmvnd.extClasspath=",
		"-Dmvnd.coreExtensionsDiscriminator=da39a3ee5e6b4b0d3255bfef95601890afd80709",
		"-Dmvnd.coreExtensionsExclude=io.takari.maven:takari-smart-builder",
		"-Dmvnd.enableAssertions=false",
		"-Dmvnd.expirationCheckDelay=10s",
		"-Dmvnd.duplicateDaemonGracePeriod=10s",
		"-Dmvnd.socketFamily=inet",
		"org.codehaus.plexus.classworlds.launcher.Launcher",
	}, " ")

	assert.Equal(t, cmdlineToCommand(commandline, nil), "Launcher")
}

func TestPrettifyJavaClass(t *testing.T) {
	assert.Equal(t, *prettifyFullyQualifiedJavaClass("com.example.MyClass"), "MyClass")
	assert.Equal(t, *prettifyFullyQualifiedJavaClass("com.example.Main"), "example.Main")
	assert.Equal(t, prettifyFullyQualifiedJavaClass(""), nil)
}
