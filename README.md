[![Linux CI
Status](https://github.com/walles/px/actions/workflows/linux-ci.yml/badge.svg)](https://github.com/walles/px/actions/workflows/linux-ci.yml?query=branch%3Apython)
[![macOS CI
Status](https://github.com/walles/px/actions/workflows/macos-ci.yml/badge.svg)](https://github.com/walles/px/actions/workflows/macos-ci.yml?query=branch%3Apython)

# `ps`, `top` and `pstree` for Human Beings

See below for [how to install](#installation).

`ptop` is what I usually use when
[LoadViz](https://github.com/walles/loadviz/) shows something unexpected
is going on.

`px` I use for figuring out things like "do I still have any
[Flutter](https://flutter.dev) processes running in the background"?

`pxtree` can be used as `watch --color pxtree brew` to figure out what
[Homebrew](https://brew.sh) is doing.

# Output

## `ptop`

If you're coming from `htop` or some other `top` variant, here's what
to expect from `ptop`, with explanations below the screenshot:

![ptop screenshot](doc/ptop-screenshot.png)

- Note the core count right next to the system load number, for easy
  comparison.
- Note the load history graph next to the load numbers. On this system
  the load has been high for the last 15 minutes. This is a
  visualization of the numbers you get from `uptime`.
- Note the bars showing which programs / users are using your memory
  below the memory numbers
- Note the `IO Load` number, showing which IO device had the highest
  average throughput since `ptop` launched.
- Note how the default sort order of CPUTIME-since-`ptop`-started makes
  the display mostly stable and enables you to sort by CPU usage.
- Note that binaries launched while `ptop` is running are listed at the
  bottom of the display.
- Note how the Python program on the second to last line is shown as
  `run_adapter.py` (the program) rather than `python3` (the runtime).
  [This support is available for many
  VMs](https://github.com/walles/px/blob/python/tests/px_commandline_test.py)
  like Java, Node, \...
- Selecting a process with Enter will offer you to see detailed
  information about that process, in `$PAGER`,
  [moor](https://github.com/walles/moor) or `less`. Or to kill it.
- After you press `q` to quit, the display is retained and some lines at
  the bottom are removed to prevent the information you want from
  scrolling out of view.
- A help text on the bottom hints you how to search / filter
  (interactively), change sort order or how to pick processes for
  further inspection or killing.

## `pxtree`

![pxtree screenshot](doc/pxtree-screenshot.png)

- Note how search hits are highlighted in **bold**
- Note how PIDs (process IDs) are printed by default
- Note how multiple processes with the same names are coalesced and
  printed with the count in parentheses
- Note how the process names make sense (`lsp_server.py` rather than
  `python3`)

## `px`

Running just `px` lists all running processes, with the most interesting
ones last. Output truncated for brevity.

    PID COMMAND                           USERNAME           CPU CPUTIME RAM COMMANDLINE
      0 kernel                            root                --      --  -- kernel PID 0
    273 SandboxHelper                     _coreaudiod         0%   0.01s  0% /System/Library/Frameworks/AudioToolbox.framework/XPCServices/com.apple.audio.SandboxHelper.xpc/Contents/MacOS/com.apple.audio.SandboxHelper
    596 installerdiagd                    root                0%   0.01s  0% /System/Library/PrivateFrameworks/InstallerDiagnostics.framework/Versions/A/Resources/installerdiagd
    983 periodic-wrapper                  root                0%   0.01s  0% /usr/libexec/periodic-wrapper daily
      ...
    57417 Google Chrome Helper              johan               0%   1m03s  2% /Applications/Google Chrome.app/Contents/Versions/70.0.3538.102/Google Chrome Helper.app/Contents/MacOS/Google Chrome Helper --type=renderer --field-trial-handle=5536258455526146518,14669732848005555331,131072 --service-pipe-token=7224348701576210538 --lang=sv --metrics-client-id=576E1A60-CA59-34F4-6C0C-57F64BD5F01C --enable-offline-auto-reload --enable-offline-auto-reload-visible-only --num-raster-threads=4 --enable-zero-copy --enable-gpu-memory-buffer-compositor-resources --enable-main-frame-before-activation --service-request-channel-token=7224348701576210538 --renderer-client-id=1119 --no-v8-untrusted-code-mitigations --seatbelt-client=418
    14983 studio                            johan               0%   1h22m 14% /Applications/Android Studio.app/Contents/MacOS/studio
    57993 kcm                               root                0%   0.02s  0% /System/Library/PrivateFrameworks/Heimdal.framework/Helpers/kcm --launchd
    57602 Code Helper                       johan               0%  12.73s  2% /private/var/folders/cg/d7qzk4s13s9c8t49t3txdjpr0000gn/T/AppTranslocation/B5DDDD81-5A91-4961-B18B-20DAB3925EB0/d/Visual Studio Code.app/Contents/Frameworks/Code Helper.app/Contents/MacOS/Code Helper --type=renderer --js-flags=--nolazy --no-sandbox --primordial-pipe-token=570B948A976AACDA8EBB532E5680C83E --lang=sv --app-path=/private/var/folders/cg/d7qzk4s13s9c8t49t3txdjpr0000gn/T/AppTranslocation/B5DDDD81-5A91-4961-B18B-20DAB3925EB0/d/Visual Studio Code.app/Contents/Resources/app --node-integration=true --webview-tag=true --no-sandbox --background-color=#171717 --disable-blink-features=Auxclick --enable-pinch --num-raster-threads=4 --enable-zero-copy --enable-gpu-memory-buffer-compositor-resources --enable-main-frame-before-activation --content-image-texture-target=0,0,3553;0,1,3553;0,2,3553;0,3,3553;0,4,3553;0,5,3553;0,6,3553;0,7,3553;0,8,3553;0,9,3553;0,10,34037;0,11,34037;0,12,34037;0,13,3553;0,14,3553;0,15,3553;1,0,3553;1,1,3553;1,2,3553;1,3,3553;1,4,3553;1,5,3553;1,6,3553;1,7,3553;1,8,3553;1,9,3553;1,10,34037;1,11,34037;1,12,34037;1,13,3553;1,14,3553;1,15,3553;2,0,3553;2,1,3553;2,2,3553;2,3,3553;2,4,3553;2,5,3553;2,6,3553;2,7,3553;2,8,3553;2,9,3553;2,10,34037;2,11,34037;2,12,34037;2,13,3553;2,14,3553;2,15,3553;3,0,3553;3,1,3553;3,2,3553;3,3,3553;3,4,3553;3,5,34037;3,6,3553;3,7,3553;3,8,3553;3,9,3553;3,10,3553;3,11,3553;3,12,34037;3,13,3553;3,14,34037;3,15,34037;4,0,3553;4,1,3553;4,2,3553;4,3,3553;4,4,3553;4,5,34037;4,6,3553;4,7,3553;4,8,3553;4,9,3553;4,10,3553;4,11,3553;4,12,34037;4,13,3553;4,14,34037;4,15,34037 --service-request-channel-token=570B948A976AACDA8EBB532E5680C83E --renderer-client-id=110
    57996 cat                               johan               0%    0.0s  0% cat
    57745 GradleDaemon                      johan               0%  32.75s  3% /Library/Java/JavaVirtualMachines/jdk1.8.0_60.jdk/Contents/Home/bin/java -Xmx1536m -Dfile.encoding=UTF-8 -Duser.country=SE -Duser.language=sv -Duser.variant -cp /Users/johan/.gradle/wrapper/dists/gradle-4.6-all/bcst21l2brirad8k2ben1letg/gradle-4.6/lib/gradle-launcher-4.6.jar org.gradle.launcher.daemon.bootstrap.GradleDaemon 4.6

- To give you the most interesting processes close to your next prompt,
  `px` puts last in its output processes that:
  - Have been started recently (can be seen in the list as high PIDs)
  - Are using lots of memory
  - Have used lots of CPU time
- Java processes are presented as their main class (`GradleDaemon`)
  rather than as their executable (`java`). [This support is available
  for many
  VMs](https://github.com/walles/px/blob/python/tests/px_commandline_test.py).

## `px java`

This lists all Java processes. Note how they are presented as their main
class (`GradleDaemon`) rather than as their executable (`java`). [This
support is available for many
VMs](https://github.com/walles/px/blob/python/tests/px_commandline_test.py).

    PID COMMAND      USERNAME CPU CPUTIME RAM COMMANDLINE
    57745 GradleDaemon johan     0%  35.09s  3% /Library/Java/JavaVirtualMachines/jdk1.8.0_60.jdk/Contents/Home/bin/java -Xmx1536m -Dfile.encoding=UTF-8 -Duser.country=SE -Dus

## `px _coreaudiod`

This lists all processes owned by the `_coreaudiod` user.

    PID COMMAND       USERNAME    CPU CPUTIME RAM COMMANDLINE
    273 SandboxHelper _coreaudiod  0%   0.01s  0% /System/Library/Frameworks/AudioToolbox.framework/XPCServices/com.apple.audio.SandboxHelper.xpc/Contents/MacOS/com.apple.audio.SandboxHelper
    190 DriverHelper  _coreaudiod  0%    0.3s  0% /System/Library/Frameworks/CoreAudio.framework/Versions/A/XPCServices/com.apple.audio.DriverHelper.xpc/Contents/MacOS/com.apple.audio.DriverHelper
    182 coreaudiod    _coreaudiod  0%  11m28s  0% /usr/sbin/coreaudiod

## `sudo px 80727`

This shows detailed info about PID 80727.

    /Library/Java/JavaVirtualMachines/jdk1.8.0_60.jdk/Contents/Home/bin/java
      -Xmx1536M
      -Dfile.encoding=UTF-8
      -Duser.country=SE
      -Duser.language=sv
      -Duser.variant
      -cp
      /Users/johan/.gradle/wrapper/dists/gradle-3.5-all/7s64ktr9gh78lhv83n6m1hq9u6/gradle-3.5/lib/gradle-launcher-3.5.jar
      org.gradle.launcher.daemon.bootstrap.GradleDaemon
      3.5

    kernel(0)                root
      launchd(1)             root
    --> GradleDaemon(80727)  johan

    31m33s ago GradleDaemon was started, at 2017-06-18T13:47:53+02:00.
    7.6% has been its average CPU usage since then, or 2m22s/31m33s

    Other processes started close to GradleDaemon(80727):
      -fish(80678) was started 9.0s before GradleDaemon(80727)
      iTerm2(80676) was started 9.0s before GradleDaemon(80727)
      login(80677) was started 9.0s before GradleDaemon(80727)
      mdworker(80729) was started just after GradleDaemon(80727)
      mdworker(80776) was started 21.0s after GradleDaemon(80727)

    Users logged in when GradleDaemon(80727) started:
      _mbsetupuser
      johan

    2017-06-18T14:19:26.521988: Now invoking lsof, this can take over a minute on a big system...
    2017-06-18T14:19:27.070396: lsof done, proceeding.

    Others sharing this process' working directory (/)
      Working directory too common, never mind.

    File descriptors:
      stdin : [PIPE] <not connected> (0x17d7619d3ae04819)
      stdout: [CHR] /dev/null
      stderr: [CHR] /dev/null

    Network connections:
      [IPv6] *:56789 (LISTEN)
      [IPv6] *:62498 (LISTEN)

    Inter Process Communication:
      mDNSResponder(201): [unix] ->0xe32cbd7be6021f1f

    For a list of all open files, do "sudo lsof -p 80727", or "sudo watch lsof -p 80727" for a live view.

- The command line has been split with one argument per line. This makes
  long command lines readable.
- The process tree shows how the Gradle Daemon relates to other
  processes.
- Details on how long ago Gradle Daemon was started, and how much CPU it
  has been using since.
- A list of other processes started around the same time as Gradle
  Daemon.
- A section describing where the standard file descriptors of the
  process go.
- A list of users logged in when the Gradle Daemon was started.
- A list of other processes with the same working directory as this one.
- A list of network connections the process has open.
- The IPC section shows that the Gradle Daemon is talking to
  `mDNSResponder` using [Unix domain
  sockets](https://en.wikipedia.org/wiki/Unix_domain_socket).

The IPC data comes from `lsof`. `sudo` helps `lsof` get more detailed
information; the command will work without it but might miss some
information.

## Killing processes

If you want an interactive process killer using `px` and
[fzf](https://github.com/junegunn/fzf), you can make a shell alias out
of this:

    px --sort=cpupercent | fzf --bind 'ctrl-r:reload(px --sort=cpupercent)' --height=20 --no-hscroll --tac --no-sort --header-lines=1 | awk '{print $1}' | xargs kill -9

Type for fuzzy process search, use arrow keys to pick a process, Enter
to kill, CTRL-R to refresh the process list.

Or with previews for the currently selected process:

    px --sort=cpupercent --no-username | fzf --preview='px --color {1}' --bind 'ctrl-r:reload(px --sort=cpupercent --no-username)' --height=20 --no-hscroll --tac --no-sort --header-lines=1 | awk '{print $1}' | xargs kill -9

## Installation

FIXME: Update since the Go rewrite

## Usage

Just type `px` or `ptop`, that's a good start!

To exit `ptop`, press `q`.

Also try `px --help` to see what else `px` can do except for just
listing all processes.

If you run into problems, try running with the `--debug` switch, it will
print debug logging output after `px`/`ptop` is done.

# Use Cases

- Why is my fan making noises?
  - Process top list
- I have a CPU meter that is peaking, why?
  - Process top list
- I have a RAM meter that is peaking, why?
  - Process top list
- Why is my computer slow?
  - Process top list
  - Process top list by IO usage
- Which processes are IO heavy?
  - Process top list by IO usage
- Is this specific process leaking memory?
  - When a process is selected, replace the user top lists with a braille
    history chart for the current process. This means we need to collect
    historical data for each process.
- Which new processes are being launched and why?
  - The ptop launched-binaries tree is excellent for this
- Is some particular service running?
  - Process search by name or number
- Which users are consuming CPU?
  - User top list by CPU usage
- Which users are consuming RAM?
  - User top list by RAM usage
- Which users are consuming IO?
  - User top list by IO usage
- I want to see the overall system load and resource usage
  - System load graph for CPU.
  - Memory pressure as measured by "system" CPU time. Or some number, since even
    if it doesn't help them, this is the number people expect to see.
  - Some IO load number.
- I need to check if my system is under heavy I/O load
  - Process top list by IO usage
  - Or if that's not possible, device top list by IO usage
- I want to see if a process is stuck or in an uninterruptible sleep state
  - Nah, let's just not care about this until somebody explicitly asks for it
- I need to find and kill a runaway process.
  - Find: Process top list
  - Kill: Select process and provide a way for the user to request its termination
- Why is some process running on my system?
  - The px-for-one-process view is excellent for this

# Development

FIXME: Update since the Go rewrite

## Releasing a new Version

FIXME: Update since the Go rewrite

## TODO

- Be happy enough with --version output
- Decide on the new name
- Rename
- Check any mention of px, ptop or pxtree is intentional
- Implement some crash reporting system
- Be happy enough with --help output
- Set up CI building + testing on Linux and at least cross compiling to macOS
- Document in this README how to make releases
- Update screenshot(s) ^
- Verify all descriptions in this file + screenshots match the actual behaviors
  of our binaries.
- Make a release.
- Profile and see if there's any low-hanging fruit to fix performance-wise
- Accept smaller window sizes
  - Drop columns if the terminal is really narrow
- Move macOS specific parsers into cross-platform parser files and add tests for
  them, just like we have for the Linux specific parsers.
- Implement filtering
- Implement process picking with arrow keys
- When hovering a process, replace the two rightmost panes with info about that
  process
- Implement the I-picked-a-process-by-pressing-enter menu screen. By spawning
  `px`?
- Verify we have all Use Cases ^ covered
- Should we remake `px`?
- Should we remake `pxtree`?
- Make sure we are implementing the memory leak use case.

## TODO `iotop` replacement

- When given the `--top` flag and enough permissions, record per process
  IO usage and present that in one or more columns.

## TODO misc

- Details: When no users were found to be logged in at process start,
  automatically detect whether it's because we don't have history that
  far back or whether it seems to be that nobody was actually logged in.
  Inform the user about the outcome.
- In the px / top views, in the process owner column, maybe print other
  non-root process owners of parent processes inside parentheses?
- Ignore -E switch on Python command lines

## DONE

- Don't crash on too small terminal window
- Get the UI structure right with plain text only. Pressing ESC / q should exit.
- Drop the two rightmost panes if the terminal is too narrow
- Drop the launched-binaries tree if the terminal is too low
