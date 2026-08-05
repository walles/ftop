package processes

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"

	"github.com/walles/ftop/internal/assert"
)

// How macOS names the two ends of an anonymous pipe: each end by its own kernel
// address in the device field, and by its peer's in the name. No inode, and no
// access mode either, macOS lsof having nothing to say about which end writes.
func TestLsofPipeParser_macOsPipePair(t *testing.T) {
	parser := newLsofPipeParser()

	lines := []string{
		"p36143\x00",
		"f1\x00a \x00tPIPE\x00d0x77046c8deffe9dd1\x00n->0x652aa8d44c539286\x00",
		"p36144\x00",
		"f0\x00a \x00tPIPE\x00d0x652aa8d44c539286\x00n->0x77046c8deffe9dd1\x00",
	}
	for _, line := range lines {
		assert.Equal(t, parser.parseLine(line), nil)
	}

	assert.Equal(t, len(parser.pipeEndsByPid), 2)
	assert.SlicesEqual(t, parser.pipeEndsByPid[36143], []PipeEnd{
		{Fd: "1", Device: "0x77046c8deffe9dd1", PeerDevice: "0x652aa8d44c539286"},
	})
	assert.SlicesEqual(t, parser.pipeEndsByPid[36144], []PipeEnd{
		{Fd: "0", Device: "0x652aa8d44c539286", PeerDevice: "0x77046c8deffe9dd1"},
	})
}

// How Linux names the two ends of an anonymous pipe: both by the pipe's inode,
// with the access modes telling them apart. The name is the literal string
// "pipe", which identifies nothing and is dropped.
//
// Every anonymous pipe on the machine lives on pipefs and so reports the same
// file system device, 0xe here, which is why the inode is what tells two of them
// apart. The device only starts saying something for a named FIFO.
func TestLsofPipeParser_linuxPipePair(t *testing.T) {
	parser := newLsofPipeParser()

	lines := []string{
		"p1234\x00",
		"f1\x00aw\x00tFIFO\x00D0xe\x00i16466\x00npipe\x00",
		"p5678\x00",
		"f0\x00ar\x00tFIFO\x00D0xe\x00i16466\x00npipe\x00",
	}
	for _, line := range lines {
		assert.Equal(t, parser.parseLine(line), nil)
	}

	assert.SlicesEqual(t, parser.pipeEndsByPid[1234], []PipeEnd{
		{Fd: "1", Access: PipeAccessWrite, FileSystemDevice: "0xe", Inode: "16466"},
	})
	assert.SlicesEqual(t, parser.pipeEndsByPid[5678], []PipeEnd{
		{Fd: "0", Access: PipeAccessRead, FileSystemDevice: "0xe", Inode: "16466"},
	})
}

// Two named FIFOs on different file systems can share an inode number, and then
// the file system device is the only thing telling them apart. Both of these are
// inode 2, each being the first file made on a freshly mounted tmpfs.
//
// lsof hands that device over under the "D" field descriptor and only under that
// one: the lowercase "d" that carries a macOS pipe's kernel address is empty for
// every FIFO, on both platforms.
func TestLsofPipeParser_linuxNamedFifosSharingAnInode(t *testing.T) {
	parser := newLsofPipeParser()

	lines := []string{
		"p188\x00",
		"f3\x00au\x00tFIFO\x00D0x37\x00i2\x00n/mnt/a/f\x00",
		"f4\x00au\x00tFIFO\x00D0x38\x00i2\x00n/mnt/b/f\x00",
	}
	for _, line := range lines {
		assert.Equal(t, parser.parseLine(line), nil)
	}

	assert.SlicesEqual(t, parser.pipeEndsByPid[188], []PipeEnd{
		{Fd: "3", Access: PipeAccessReadWrite, FileSystemDevice: "0x37", Inode: "2"},
		{Fd: "4", Access: PipeAccessReadWrite, FileSystemDevice: "0x38", Inode: "2"},
	})
}

// A named FIFO is a FIFO with a path, reported with an inode and an access mode
// on both platforms. This one is from macOS, which spells anonymous pipes
// entirely differently, so the two kinds have to both come through.
//
// The path is no peer, however much of a name it is, and must not be mistaken
// for one.
func TestLsofPipeParser_namedFifo(t *testing.T) {
	parser := newLsofPipeParser()

	lines := []string{
		"p36166\x00",
		"f4\x00ar\x00tFIFO\x00i82144503\x00n/private/tmp/probe.fifo\x00",
		"f5\x00au\x00tFIFO\x00i82144503\x00n/private/tmp/probe.fifo\x00",
	}
	for _, line := range lines {
		assert.Equal(t, parser.parseLine(line), nil)
	}

	assert.SlicesEqual(t, parser.pipeEndsByPid[36166], []PipeEnd{
		{Fd: "4", Access: PipeAccessRead, Inode: "82144503"},
		{Fd: "5", Access: PipeAccessReadWrite, Inode: "82144503"},
	})
}

// Pipes have no filter flag of their own, so this listing arrives mixed in with
// every other open file on the machine. Everything that isn't a pipe has to go,
// and must not cost us the pipes around it.
//
// The unix domain socket is the one to watch: macOS names those exactly the way
// it names pipes, so the type is all that keeps them apart.
func TestLsofPipeParser_ignoresOtherFileTypes(t *testing.T) {
	parser := newLsofPipeParser()

	lines := []string{
		"p36143\x00",
		"fcwd\x00a \x00tDIR\x00i72202814\x00n/private/tmp\x00",
		"ftxt\x00a \x00tREG\x00i1152921500312522712\x00n/bin/sleep\x00",
		"f0\x00ar\x00tCHR\x00i314\x00n/dev/null\x00",
		"f1\x00a \x00tPIPE\x00d0x77046c8deffe9dd1\x00n->0x652aa8d44c539286\x00",
		"f2\x00au\x00tunix\x00d0x98e949eb4ef84f40\x00n->0x74cc3fffa76a0c30\x00",
		"f3\x00au\x00tIPv4\x00d0x8df6312a08cb9a23\x00n192.168.50.32:54599->172.217.19.234:443\x00",
		"f4\x00au\x00tKQUEUE\x00ncount=0, state=0x12\x00",
	}
	for _, line := range lines {
		assert.Equal(t, parser.parseLine(line), nil)
	}

	assert.SlicesEqual(t, parser.pipeEndsByPid[36143], []PipeEnd{
		{Fd: "1", Device: "0x77046c8deffe9dd1", PeerDevice: "0x652aa8d44c539286"},
	})
}

// lsof reports a pipe whose other end is gone with no name at all. There is
// nobody to match it against, but it is still a pipe end, and reporting it as
// one keeps the decision about what to do with it in PipeConnections().
func TestLsofPipeParser_pipeWithoutAPeer(t *testing.T) {
	parser := newLsofPipeParser()

	assert.Equal(t, parser.parseLine("p36180\x00"), nil)
	assert.Equal(t, parser.parseLine("f1\x00a \x00tPIPE\x00d0xc6eabd04a1b56e60\x00n\x00"), nil)

	assert.SlicesEqual(t, parser.pipeEndsByPid[36180], []PipeEnd{
		{Fd: "1", Device: "0xc6eabd04a1b56e60"},
	})
}

// Pipe ends belong to the process lsof most recently named, so one arriving
// before any process at all is something we can't make sense of.
func TestLsofPipeParser_pipeBeforeAnyPid(t *testing.T) {
	parser := newLsofPipeParser()

	err := parser.parseLine("f1\x00a \x00tPIPE\x00d0x77046c8deffe9dd1\x00n->0x652aa8d44c539286\x00")

	assert.Equal(t, err == nil, false)
}

func TestLsofPipeParser_unparseablePid(t *testing.T) {
	parser := newLsofPipeParser()

	err := parser.parseLine("pnotanumber\x00")

	assert.Equal(t, err == nil, false)
}

// The real lsof should see a pipe we just made ourselves, both ends of it, and
// tell us enough about each end to match the two against each other.
//
// The two ways of matching are keyed on the peer's device and on the inode
// respectively, and this is what says that this platform's lsof reports at least
// one of them for a pipe. Without that the matching has nothing to work with,
// however correct it is.
func TestGetPipeEndsByPid(t *testing.T) {
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("lsof not available: ", err)
	}

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
	}()

	readerFd := strconv.Itoa(int(reader.Fd()))
	writerFd := strconv.Itoa(int(writer.Fd()))

	pipeEndsByPid, err := GetPipeEndsByPid()
	if err != nil {
		t.Fatalf("listing pipes failed: %v", err)
	}

	foundReader := false
	foundWriter := false
	for _, end := range pipeEndsByPid[os.Getpid()] {
		if end.Fd != readerFd && end.Fd != writerFd {
			// Some other pipe of ours, and those come in shapes this says
			// nothing about: one whose other end is gone carries neither of the
			// two fields below.
			continue
		}

		// One or the other, never neither: macOS names an anonymous pipe end by
		// its peer's device and gives it no inode, Linux the other way around.
		assert.Equal(t, end.PeerDevice != "" || end.Inode != "", true)

		if end.Fd == readerFd {
			foundReader = true
		}

		if end.Fd == writerFd {
			foundWriter = true
		}
	}

	assert.Equal(t, foundReader, true)
	assert.Equal(t, foundWriter, true)
}

// The real lsof should tell us enough about the two ends of a named FIFO for
// arePipeEnds() to match them, on whatever platform and lsof version this is.
//
// A FIFO is the one pipe both platforms report an inode for, and the one Linux
// reports a file system device for, so the two matching clauses meet here: the
// inode has to be there, and whatever device comes along with it must not keep
// two ends of one and the same FIFO apart.
func TestGetPipeEndsByPid_namedFifo(t *testing.T) {
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("lsof not available: ", err)
	}

	path := filepath.Join(t.TempDir(), "probe.fifo")
	err := syscall.Mkfifo(path, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	// Read-write first: opening a FIFO that way never blocks, and it makes the
	// reader below a reader of a FIFO somebody already has open for writing,
	// which doesn't block either.
	readWrite, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = readWrite.Close() }()

	read, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = read.Close() }()

	readWriteFd := strconv.Itoa(int(readWrite.Fd()))
	readFd := strconv.Itoa(int(read.Fd()))

	pipeEndsByPid, err := GetPipeEndsByPid()
	if err != nil {
		t.Fatalf("listing pipes failed: %v", err)
	}

	var readWriteEnd *PipeEnd
	var readEnd *PipeEnd
	for _, end := range pipeEndsByPid[os.Getpid()] {
		switch end.Fd {
		case readWriteFd:
			readWriteEnd = &end

		case readFd:
			readEnd = &end
		}
	}

	if readWriteEnd == nil || readEnd == nil {
		t.Fatalf("lsof reported %v, missing fd %s or %s",
			pipeEndsByPid[os.Getpid()], readWriteFd, readFd)
	}

	// The inode is what identifies a FIFO on both platforms, and lsof reporting
	// the access modes is what tells its ends apart.
	assert.Equal(t, readWriteEnd.Inode != "", true)
	assert.Equal(t, readWriteEnd.Access, PipeAccessReadWrite)
	assert.Equal(t, readEnd.Access, PipeAccessRead)

	assert.Equal(t, arePipeEnds(*readWriteEnd, *readEnd), true)
}
