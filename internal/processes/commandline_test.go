package processes

import (
	"os"
	"path"
	"path/filepath"
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

func TestGetTerraformProviderCommandline(t *testing.T) {
	// Source: https://github.com/walles/px/issues/105
	assert.Equal(t,
		cmdlineToCommand(".terraform/providers/registry.terraform.io/heroku/heroku/4.8.0/darwin_amd64/terraform-provider-heroku_v4.8.0"),
		"terraform-provider-heroku_v4.8.0",
	)
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

func TestGetCommandRubySwitches(t *testing.T) {
	// ruby with warning level switch and brew.rb subcommand
	assert.Equal(t,
		cmdlineToCommand("/usr/bin/ruby -W0 /usr/local/bin/brew.rb install rust"),
		"brew.rb install",
	)

	// Double-dash to end options, should pick the first script after it
	assert.Equal(t,
		cmdlineToCommand("/usr/bin/ruby -W1 -- /apa/build.rb /bepa/cmake.rb"),
		"build.rb",
	)

	// Encoding switch should be ignored
	assert.Equal(t,
		cmdlineToCommand("/usr/bin/ruby -Eascii-8bit:ascii-8bit /usr/sbin/google-fluentd"),
		"google-fluentd",
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
		}, " ")),
		"cloc",
	)

	assert.Equal(t,
		cmdlineToCommand("/usr/bin/perl /usr/local/Cellar/cloc/1.90/libexec/bin/cloc"),
		"cloc",
	)
	assert.Equal(t,
		cmdlineToCommand("perl /usr/local/Cellar/cloc/1.90/libexec/bin/cloc"),
		"cloc",
	)
	assert.Equal(t,
		cmdlineToCommand("/usr/bin/perl5 /usr/local/Cellar/cloc/1.90/libexec/bin/cloc"),
		"cloc",
	)
	assert.Equal(t,
		cmdlineToCommand("/usr/bin/perl5.30 /usr/local/Cellar/cloc/1.90/libexec/bin/cloc"),
		"cloc",
	)

	// Give up on command line switches
	assert.Equal(t,
		cmdlineToCommand("/usr/bin/perl -S cloc"),
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
	assert.Equal(t, cmdlineToCommand(dock), "Dock/extra")

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
	assert.Equal(t, cmdlineToCommand(firefox), "Firefox/plugin-container")

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
	assert.Equal(t, cmdlineToCommand(codeHelper), "CodeHelper(Renderer)")

	// iTerm without prefix (human-friendly command)
	assert.Equal(t, cmdlineToCommand("/Applications/iTerm.app/Contents/MacOS/iTerm2"), "iTerm2")

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
	assert.Equal(t, cmdlineToCommand(ids), "IDS/identityservicesd")

	intelligencePlatform := strings.Join([]string{
		"/System",
		"/Library",
		"/PrivateFrameworks",
		"/IntelligencePlatformCore.framework",
		"/Versions",
		"/A",
		"/intelligenceplatformd",
	}, "/")
	assert.Equal(t, cmdlineToCommand(intelligencePlatform), "IntelligencePlatformCore (Daemon)")
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
		}, " ")),
		"VisualStudioCode",
	)
}

func TestGetCommandFlutter(t *testing.T) {
	// Subcommand without path/dots -> treat as subcommand
	assert.Equal(t,
		cmdlineToCommand(strings.Join([]string{
			"/usr/local/Cellar/dart/2.15.1/libexec/bin/dart",
			"devtools",
		}, " ")),
		"dart devtools",
	)

	// Dotted candidate -> treat as file
	assert.Equal(t,
		cmdlineToCommand(strings.Join([]string{
			"/usr/local/Cellar/dart/2.15.1/libexec/bin/dart",
			"devtools.snap",
		}, " ")),
		"devtools.snap",
	)

	// Path candidate -> basename
	assert.Equal(t,
		cmdlineToCommand(strings.Join([]string{
			"/usr/local/Cellar/dart/2.15.1/libexec/bin/dart",
			"/usr/local/bin/devtools",
		}, " ")),
		"devtools",
	)
}

func TestGetCommandGuile(t *testing.T) {
	// Plain guile
	assert.Equal(t, cmdlineToCommand("guile"), "guile")

	// -l returns its arg as the script
	assert.Equal(t,
		cmdlineToCommand("guile -l myscript.scm"),
		"myscript.scm",
	)

	// Ignore common switches
	assert.Equal(t,
		cmdlineToCommand("guile -q --r6rs myscript.scm"),
		"myscript.scm",
	)

	// Ignore argful switches
	assert.Equal(t,
		cmdlineToCommand("guile -L /usr/local/include -C /tmp -x scheme myscript.scm"),
		"myscript.scm",
	)

	// --listen with equals form
	assert.Equal(t,
		cmdlineToCommand("guile --listen=localhost:37146 myscript.scm"),
		"myscript.scm",
	)

	// Unknown switch after guile -> give up
	assert.Equal(t,
		cmdlineToCommand("guile --unknown myscript.scm"),
		"guile",
	)
}

func TestGetCommandResque(t *testing.T) {
	assert.Equal(t, cmdlineToCommand("resque-1.20.0: a b c"), "resque-1.20.0:")
	assert.Equal(t, cmdlineToCommand("resqued-0.7.12 x y z"), "resqued-0.7.12")
}

func TestGetHomebrewCommandline(t *testing.T) {
	assert.Equal(t,
		cmdlineToCommand(strings.Join([]string{
			"/usr/local/Homebrew/Library/Homebrew/vendor/portable-ruby/current/bin/ruby",
			"-W0",
			"--disable=gems,did_you_mean,rubyopt",
			"/usr/local/Homebrew/Library/Homebrew/brew.rb",
			"upgrade",
		}, " ")),
		"brew.rb upgrade",
	)
}

func TestGetCommandUnicode(t *testing.T) {
	// Emoji-only command should be preserved
	assert.Equal(t, cmdlineToCommand("😀"), "😀")

	// Simple unicode executable name
	assert.Equal(t, cmdlineToCommand("/usr/local/bin/äpple"), "äpple")

	// Python running a unicode script path -> basename
	assert.Equal(t, cmdlineToCommand("python /usr/bin/hällo.py"), "hällo.py")

	// Ruby running a unicode script path -> basename
	assert.Equal(t, cmdlineToCommand("ruby /some/path/тест.rb"), "тест.rb")

	// Shell running a unicode script path -> basename
	assert.Equal(t, cmdlineToCommand("bash /some/path/ユニコード.sh"), "ユニコード.sh")
}

func TestGetCommandJava(t *testing.T) {
	// Basics
	assert.Equal(t, cmdlineToCommand("java"), "java")
	assert.Equal(t, cmdlineToCommand("java -version"), "java")
	assert.Equal(t, cmdlineToCommand("java -help"), "java")

	// Class and jar
	assert.Equal(t, cmdlineToCommand("java SomeClass"), "SomeClass")
	assert.Equal(t, cmdlineToCommand("java x.y.SomeClass"), "SomeClass")
	assert.Equal(t, cmdlineToCommand("java -jar flaska.jar"), "flaska.jar")
	assert.Equal(t, cmdlineToCommand("java -jar /a/b/flaska.jar"), "flaska.jar")

	// Special handling of Main
	assert.Equal(t, cmdlineToCommand("java a.b.c.Main"), "c.Main")

	// Ignore certain options
	assert.Equal(t, cmdlineToCommand("java -server SomeClass"), "SomeClass")
	assert.Equal(t, cmdlineToCommand("java -Xwhatever SomeClass"), "SomeClass")
	assert.Equal(t, cmdlineToCommand("java -Dwhatever SomeClass"), "SomeClass")
	assert.Equal(t, cmdlineToCommand("java -eahej SomeClass"), "SomeClass")
	assert.Equal(t, cmdlineToCommand("java -dahej SomeClass"), "SomeClass")
	assert.Equal(t, cmdlineToCommand("java -cp /a/b/c SomeClass"), "SomeClass")
	assert.Equal(t, cmdlineToCommand("java -classpath /a/b/c SomeClass"), "SomeClass")

	// Invalid command lines
	assert.Equal(t, cmdlineToCommand("java -cp /a/b/c"), "java")
	assert.Equal(t, cmdlineToCommand("java  "), "java")
	assert.Equal(t, cmdlineToCommand("java -jar"), "java")
	assert.Equal(t, cmdlineToCommand("java -jar    "), "java")
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

	assert.Equal(t, cmdlineToCommand(commandline), "org.eclipse.equinox.launcher_1.5.800.v20200727-1323.jar")
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

	assert.Equal(t, cmdlineToCommand(commandline), "GradleDaemon")
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

	assert.Equal(t, cmdlineToCommand(commandline), "GradleWorkerMain")
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

	assert.Equal(t, cmdlineToCommand(commandline), "jruby.Main")
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

	assert.Equal(t, cmdlineToCommand(commandline), "Bootstrap")
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

	assert.Equal(t, cmdlineToCommand(commandline), "GradleWorkerMain")
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

	assert.Equal(t, cmdlineToCommand(commandline), "Launcher")
}

func TestPrettifyJavaClass(t *testing.T) {
	assert.Equal(t, *prettifyFullyQualifiedJavaClass("com.example.MyClass"), "MyClass")
	assert.Equal(t, *prettifyFullyQualifiedJavaClass("com.example.Main"), "example.Main")
	assert.Equal(t, prettifyFullyQualifiedJavaClass(""), nil)
}
