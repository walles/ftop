package processes

import (
	"os"
	"path/filepath"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/walles/ptop/internal/assert"
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

func TestCoalesceCount(t *testing.T) {
	exists := func(path string) bool {
		return slices.Contains([]string{"/", "/a b c", "/a b c/"}, path)
	}

	assert.Equal(t, coalesceCount([]string{"/a", "b", "c"}, exists), 3)
	assert.Equal(t, coalesceCount([]string{"/a", "b", "c/"}, exists), 3)
	assert.Equal(t, coalesceCount([]string{"/a", "b", "c", "d"}, exists), 3)

	assert.Equal(t,
		coalesceCount([]string{"/a", "b", "c:/a", "b", "c"}, exists),
		5,
	)
	assert.Equal(t,
		coalesceCount([]string{"/a", "b", "c/:/a", "b", "c/"}, exists),
		5,
	)

	assert.Equal(t,
		coalesceCount([]string{"/a", "b", "c:/a", "b", "c", "d"}, exists),
		5,
	)
	assert.Equal(t,
		coalesceCount([]string{"/a", "b", "c/:/a", "b", "c/", "d/"}, exists),
		5,
	)
}

func TestToSliceSpaced1(t *testing.T) {
	exists := func(path string) bool {
		return slices.Contains([]string{
			"/Applications",
			"/Applications/IntelliJ IDEA.app",
			"/Applications/IntelliJ IDEA.app/Contents",
		}, path)
	}

	result := cmdlineToSlice(
		"java -Dhello=/Applications/IntelliJ IDEA.app/Contents",
		exists,
	)

	assert.SlicesEqual(t, result, []string{
		"java",
		"-Dhello=/Applications/IntelliJ IDEA.app/Contents",
	})
}

func TestToSliceSpaced2(t *testing.T) {
	exists := func(path string) bool {
		return slices.Contains([]string{
			"/Applications",
			"/Applications/IntelliJ IDEA.app/Contents/Info.plist",
			"/Applications/IntelliJ IDEA.app/Contents/plugins/maven-model/lib/maven-model.jar",
			"/Applications/IntelliJ IDEA.app/Contents/plugins/maven-server/lib/maven-server.jar",
			"/Applications/IntelliJ IDEA.app/Contents/plugins/maven/lib/maven3-server-common.jar",
		}, path)
	}

	result := cmdlineToSlice(strings.Join([]string{
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
	exists := func(path string) bool {
		return slices.Contains([]string{
			"/Applications",
			"/Applications/IntelliJ IDEA CE.app/Contents/Info.plist",
			"/Applications/IntelliJ IDEA CE.app/Contents/plugins/maven-model/lib/maven-model.jar",
			"/Applications/IntelliJ IDEA CE.app/Contents/plugins/maven-server/lib/maven-server.jar",
			"/Applications/IntelliJ IDEA CE.app/Contents/plugins/maven/lib/maven3-server-common.jar",
		}, path)
	}

	result := cmdlineToSlice(strings.Join([]string{
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

	existsList := []string{}
	partial := complete
	for {
		existsList = append(existsList, partial)
		partial = path.Dir(partial)
		if partial == "/" {
			break
		}
	}

	exists := func(path string) bool {
		return slices.Contains(existsList, path)
	}

	result := cmdlineToSlice(complete+" --type=gpu-process", exists)

	assert.SlicesEqual(t, result, []string{complete, "--type=gpu-process"})
}

func TestDotnetCommandline(t *testing.T) {
	// Unclear whether "fable" is a builtin or a separate tool, go with "dotnet fable".
	assert.Equal(t,
		cmdlineToCommand("libexec/dotnet fable core/Core.Test.fsproj"),
		"dotnet fable",
	)

	// The DLL has a path, so it can't be a builtin. Go with just "fable.dll"
	assert.Equal(t,
		cmdlineToCommand("libexec/dotnet any/fable.dll core/Core.Test.fsproj"),
		"fable.dll",
	)
}

func TestNodeMaxOldSpace(t *testing.T) {
	assert.Equal(t,
		cmdlineToCommand("node --max_old_space_size=4096 scripts/start.js"),
		"start.js",
	)
}

func TestGetBashBrewShCommandline(t *testing.T) {
	assert.Equal(t,
		cmdlineToCommand("/bin/bash -p /usr/local/Homebrew/Library/Homebrew/brew.sh upgrade"),
		"brew.sh upgrade",
	)
}

func TestGetCommandInterpreters(t *testing.T) {
	// ruby
	assert.Equal(t, cmdlineToCommand("ruby"), "ruby")
	assert.Equal(t, cmdlineToCommand("ruby /some/path/apa.rb"), "apa.rb")
	assert.Equal(t, cmdlineToCommand("ruby -option /some/path/apa.rb"), "ruby")

	// sh
	assert.Equal(t, cmdlineToCommand("sh"), "sh")
	assert.Equal(t, cmdlineToCommand("sh /some/path/apa.sh"), "apa.sh")
	assert.Equal(t, cmdlineToCommand("sh -option /some/path/apa.sh"), "sh")

	// bash
	assert.Equal(t, cmdlineToCommand("bash"), "bash")
	assert.Equal(t, cmdlineToCommand("bash /some/path/apa.sh"), "apa.sh")
	assert.Equal(t, cmdlineToCommand("bash -option /some/path/apa.sh"), "bash")

	// perl
	assert.Equal(t, cmdlineToCommand("perl"), "perl")
	assert.Equal(t, cmdlineToCommand("perl /some/path/apa.pl"), "apa.pl")
	assert.Equal(t, cmdlineToCommand("perl -option /some/path/apa.pl"), "perl")
}

func TestGetGoCommandline(t *testing.T) {
	assert.Equal(t, cmdlineToCommand("go build ./..."), "go build")
	assert.Equal(t, cmdlineToCommand("go --version"), "go")
	assert.Equal(t, cmdlineToCommand("/usr/local/bin/go"), "go")
}

func TestGetGitCommandline(t *testing.T) {
	assert.Equal(t, cmdlineToCommand("git clone git@github.com:walles/riff"), "git clone")
	assert.Equal(t, cmdlineToCommand("git --version"), "git")
	assert.Equal(t, cmdlineToCommand("/usr/local/bin/git"), "git")
}

func TestGetTerraformCommandline(t *testing.T) {
	assert.Equal(t, cmdlineToCommand("terraform -chdir=dev apply -target=abc123"), "terraform apply")
}

func TestGetCommandPython(t *testing.T) {
	// Basics
	assert.Equal(t, cmdlineToCommand("python"), "python")
	assert.Equal(t, cmdlineToCommand("/apa/Python"), "Python")
	assert.Equal(t, cmdlineToCommand("python --help"), "python")

	// Running scripts / modules
	assert.Equal(t, cmdlineToCommand("python apa.py"), "apa.py")
	assert.Equal(t, cmdlineToCommand("python /usr/bin/apa.py"), "apa.py")
	assert.Equal(t, cmdlineToCommand("python2.7 /usr/bin/apa.py"), "apa.py")
	assert.Equal(t, cmdlineToCommand("python /usr/bin/hej"), "hej")
	assert.Equal(t, cmdlineToCommand("python /usr/bin/hej gris --flaska"), "hej")
	assert.Equal(t, cmdlineToCommand("python -c cmd"), "python")
	assert.Equal(t, cmdlineToCommand("python -m mod"), "mod")
	assert.Equal(t, cmdlineToCommand("python -m mod --hej gris --frukt"), "mod")
	assert.Equal(t, cmdlineToCommand("Python -"), "Python")

	// Ignoring switches
	assert.Equal(t, cmdlineToCommand("python -E apa.py"), "apa.py")
	assert.Equal(t, cmdlineToCommand("python3 -E"), "python3")
	assert.Equal(t, cmdlineToCommand("python -u -t -m mod"), "mod")

	// -W switches unsupported for now
	assert.Equal(t, cmdlineToCommand("python -W warning:spec apa.py"), "python")

	// Invalid command lines
	assert.Equal(t, cmdlineToCommand("python -W"), "python")
	assert.Equal(t, cmdlineToCommand("python -c"), "python")
	assert.Equal(t, cmdlineToCommand("python -m"), "python")
	assert.Equal(t, cmdlineToCommand("python -m   "), "python")
	assert.Equal(t, cmdlineToCommand("python -m -u"), "python")
	assert.Equal(t, cmdlineToCommand("python    "), "python")
}

func TestGetCommandAws(t *testing.T) {
	// Python wrapper around aws
	assert.Equal(t, cmdlineToCommand("Python /usr/local/bin/aws"), "aws")
	assert.Equal(t, cmdlineToCommand("python aws s3"), "aws s3")
	assert.Equal(t, cmdlineToCommand("python3 aws s3 help"), "aws s3 help")
	assert.Equal(t, cmdlineToCommand("/wherever/python3 aws s3 help flaska"), "aws s3 help")

	assert.Equal(t, cmdlineToCommand("python aws s3 sync help"), "aws s3 sync help")
	assert.Equal(t, cmdlineToCommand("python aws s3 sync nothelp"), "aws s3 sync")
	assert.Equal(t, cmdlineToCommand("python aws s3 --unknown sync"), "aws s3")

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
	assert.Equal(t, cmdlineToCommand(cmd), "aws s3 sync")
}

func TestGetCommandSudo(t *testing.T) {
	assert.Equal(t, cmdlineToCommand("sudo"), "sudo")
	assert.Equal(t, cmdlineToCommand("sudo python /usr/bin/hej gris --flaska"), "sudo hej")

	// With flags we give up and keep just "sudo"
	assert.Equal(t, cmdlineToCommand("sudo -B python /usr/bin/hej"), "sudo")
}

func TestGetCommandSudoWithSpaceInCommandName(t *testing.T) {
	dir := t.TempDir()
	spacedPath := filepath.Join(dir, "i contain spaces")
	err := os.WriteFile(spacedPath, []byte(""), 0o644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Verify splitting of the spaced file name
	assert.Equal(t, cmdlineToCommand("sudo "+spacedPath), "sudo i contain spaces")

	// Verify splitting with more parameters on the line
	assert.Equal(t, cmdlineToCommand("sudo "+spacedPath+" parameter"), "sudo i contain spaces")
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
	assert.Equal(t, cmdlineToCommand("sudo "+spacedPath), "sudo runme")

	// Verify splitting with more parameters on the line
	assert.Equal(t, cmdlineToCommand("sudo "+spacedPath+" parameter"), "sudo runme")
}
