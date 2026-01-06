package processes

import (
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
