package processes

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/walles/ftop/internal/assert"
	"golang.org/x/sys/unix"
)

// The kernel should tell us both halves of what lsof leaves out on Linux: who is
// at the other end of a connected socket, and the path it was made over.
//
// All of these sockets are ours, which the dump neither knows nor cares about —
// it reports every unix socket in our network namespace, whoever holds it.
//
// The path is what the direction is read off, so what it says about which end
// carries one is as load bearing as the peer edge itself: the socket that dialed
// has no name at all, while the socket accepted on the path is named by it. See
// unixSocketDirection().
func TestUnixSocketPeersByInode(t *testing.T) {
	// Not t.TempDir(): a unix socket path has a length limit around 100
	// characters, and a temporary directory named after this test eats into it.
	directory, err := os.MkdirTemp("", "ftop")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(directory) }()

	path := filepath.Join(directory, "probe.sock")
	client, accepted, listener := connectedUnixSocketPair(t, path)

	peersByInode, err := unixSocketPeersByInode()
	if err != nil {
		t.Fatalf("dumping unix socket peers failed: %v", err)
	}

	clientInode := socketInode(t, client)
	acceptedInode := socketInode(t, accepted)

	// The pairing is symmetric, unlike lsof's naming on macOS
	assert.Equal(t, peersByInode[clientInode], unixSocketPeer{peerInode: acceptedInode})
	assert.Equal(t, peersByInode[acceptedInode], unixSocketPeer{peerInode: clientInode, path: path})

	// A listener has a path and nobody to talk to, which is why it makes no
	// connection and gets no line
	assert.Equal(t, peersByInode[socketInode(t, listener)], unixSocketPeer{path: path})
}

// A socketpair(2) has no path on either end, which is what makes its direction
// unknown, and its two ends still name each other.
func TestUnixSocketPeersByInode_socketPair(t *testing.T) {
	pair, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = unix.Close(pair[0]) }()
	defer func() { _ = unix.Close(pair[1]) }()

	peersByInode, err := unixSocketPeersByInode()
	if err != nil {
		t.Fatalf("dumping unix socket peers failed: %v", err)
	}

	first := socketInode(t, pair[0])
	second := socketInode(t, pair[1])

	assert.Equal(t, peersByInode[first], unixSocketPeer{peerInode: second})
	assert.Equal(t, peersByInode[second], unixSocketPeer{peerInode: first})
}

// A socket in the abstract namespace comes back spelled the way lsof and
// /proc/net/unix spell it, so that a path is one path however we came by it.
func TestUnixSocketPeersByInode_abstractNamespace(t *testing.T) {
	name := "@ftop-probe"
	client, accepted, _ := connectedUnixSocketPair(t, name)

	peersByInode, err := unixSocketPeersByInode()
	if err != nil {
		t.Fatalf("dumping unix socket peers failed: %v", err)
	}

	clientInode := socketInode(t, client)
	acceptedInode := socketInode(t, accepted)

	assert.Equal(t, peersByInode[clientInode], unixSocketPeer{peerInode: acceptedInode})
	assert.Equal(t, peersByInode[acceptedInode], unixSocketPeer{peerInode: clientInode, path: name})
}

// The kernel hands out a name as it was bound, and neither of the two shapes that
// comes in is what anything else prints.
func TestUnixSocketPath(t *testing.T) {
	// A path is terminated, the terminator being part of the sockaddr
	assert.Equal(t, unixSocketPath("/tmp/probe.sock\x00"), "/tmp/probe.sock")

	// The abstract namespace is a leading NUL, spelled "@" everywhere else
	assert.Equal(t, unixSocketPath("\x00ftop-probe"), "@ftop-probe")

	// A socket bound to nothing has no name
	assert.Equal(t, unixSocketPath(""), "")
}

// A client connected to a server over path, plus the listener it was accepted on.
//
// Raw file descriptors rather than a net.Listener, so that the inodes these tests
// look sockets up by are an fstat away. A name starting with "@" binds in the
// abstract namespace, the way it does for net.Listen().
func connectedUnixSocketPair(t *testing.T, path string) (client int, accepted int, listener int) {
	t.Helper()

	address := &unix.SockaddrUnix{Name: path}

	listener, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = unix.Close(listener) })

	err = unix.Bind(listener, address)
	if err != nil {
		t.Fatal(err)
	}

	err = unix.Listen(listener, 1)
	if err != nil {
		t.Fatal(err)
	}

	client, err = unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = unix.Close(client) })

	err = unix.Connect(client, address)
	if err != nil {
		t.Fatal(err)
	}

	accepted, _, err = unix.Accept(listener)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = unix.Close(accepted) })

	return client, accepted, listener
}

// The inode of the socket on fd, spelled the way the netlink dump and lsof both
// spell it.
func socketInode(t *testing.T, fd int) string {
	t.Helper()

	var status unix.Stat_t
	err := unix.Fstat(fd, &status)
	if err != nil {
		t.Fatal(err)
	}

	return strconv.FormatUint(uint64(status.Ino), 10)
}
