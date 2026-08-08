package processes

import (
	"encoding/binary"
	"errors"
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

// One record of a dump, decoded into the inode it is about and what the dump says
// about that socket.
func TestParseUnixDiagRecord(t *testing.T) {
	record := unixDiagRecordBytes(378,
		netlinkAttributeBytes(unixDiagName, []byte("/tmp/probe.sock\x00")),
		netlinkAttributeBytes(unixDiagPeer, inodeAttributeValue(379)))

	inode, peer, err := parseUnixDiagRecord(record)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, inode, "378")
	assert.Equal(t, peer, unixSocketPeer{peerInode: "379", path: "/tmp/probe.sock"})
}

// Attributes start on four byte boundaries, so one whose value does not end on
// one is followed by padding that belongs to nobody. Reading the next attribute
// from the wrong offset would make nonsense of the rest of the record, and an
// abstract name of this length is what puts padding between the two.
func TestParseUnixDiagRecord_paddedName(t *testing.T) {
	record := unixDiagRecordBytes(378,
		netlinkAttributeBytes(unixDiagName, []byte("\x00ftop")),
		netlinkAttributeBytes(unixDiagPeer, inodeAttributeValue(379)))

	inode, peer, err := parseUnixDiagRecord(record)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, inode, "378")
	assert.Equal(t, peer, unixSocketPeer{peerInode: "379", path: "@ftop"})
}

// A socket bound to nothing that nobody is connected to: the dump reports it,
// with neither of the attributes we asked for, and there is nothing to say about
// it beyond its inode.
func TestParseUnixDiagRecord_noAttributes(t *testing.T) {
	inode, peer, err := parseUnixDiagRecord(unixDiagRecordBytes(378))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, inode, "378")
	assert.Equal(t, peer, unixSocketPeer{})
}

// A record too short to hold even the fixed size head is one we can make nothing
// of, inode included, so it fails rather than reporting a socket we invented.
func TestParseUnixDiagRecord_shortRecord(t *testing.T) {
	_, _, err := parseUnixDiagRecord(unixDiagRecordBytes(378)[:8])
	if err == nil {
		t.Fatal("expected a record shorter than the head to fail")
	}
}

// A peer attribute is an inode, and one too short to hold a four byte inode fails
// rather than being read past its end.
func TestParseUnixDiagRecord_shortPeer(t *testing.T) {
	record := unixDiagRecordBytes(378, netlinkAttributeBytes(unixDiagPeer, []byte{1, 2}))

	_, _, err := parseUnixDiagRecord(record)
	if err == nil {
		t.Fatal("expected a peer attribute shorter than an inode to fail")
	}
}

// The error an NLMSG_ERROR message carries is a negative errno, and comes back as
// the errno itself so that a caller can match on it.
func TestNetlinkError(t *testing.T) {
	// A variable rather than an expression: the kernel writes a negative errno
	// here, which is not a constant Go will convert to unsigned for us
	negativeErrno := -int32(unix.EACCES)

	err := netlinkError(binary.NativeEndian.AppendUint32(nil, uint32(negativeErrno)))
	if !errors.Is(err, unix.EACCES) {
		t.Fatalf("got %v, expected it to be EACCES", err)
	}
}

// An error message too short to hold an errno still has to fail, there being no
// success to report either way.
func TestNetlinkError_short(t *testing.T) {
	err := netlinkError([]byte{1, 2})
	if err == nil {
		t.Fatal("expected a truncated error message to fail")
	}
}

// A truncated attribute ends the list rather than failing it: the attributes
// before it parsed, and are worth as much as they ever were.
func TestNetlinkAttributes_truncated(t *testing.T) {
	first := netlinkAttributeBytes(unixDiagName, []byte("/tmp/probe.sock\x00"))
	second := netlinkAttributeBytes(unixDiagPeer, inodeAttributeValue(379))

	// Everything but the last byte of the peer attribute
	data := append(first, second[:len(second)-1]...)

	attributes := netlinkAttributes(data)

	assert.Equal(t, len(attributes), 1)
	assert.Equal(t, attributes[0].kind, uint16(unixDiagName))
	assert.Equal(t, string(attributes[0].value), "/tmp/probe.sock\x00")
}

// An attribute claiming to be shorter than its own header describes no value at
// all, and ends the list the way any other unreadable one does.
func TestNetlinkAttributes_impossibleLength(t *testing.T) {
	data := binary.NativeEndian.AppendUint16(nil, 2)
	data = binary.NativeEndian.AppendUint16(data, unixDiagName)

	assert.Equal(t, len(netlinkAttributes(data)), 0)
}

// One unix_diag_msg record as a dump lays it out: the fixed size head naming the
// socket by inode, and then the attributes the dump was asked for.
func unixDiagRecordBytes(inode uint32, attributes ...[]byte) []byte {
	record := []byte{unix.AF_UNIX, unix.SOCK_STREAM, 0 /* state */, 0 /* pad */}
	record = binary.NativeEndian.AppendUint32(record, inode)
	record = binary.NativeEndian.AppendUint32(record, 0)
	record = binary.NativeEndian.AppendUint32(record, 0)

	for _, attribute := range attributes {
		record = append(record, attribute...)
	}

	return record
}

// One netlink attribute, the TLV struct rtattr heads, padded out to the four byte
// boundary the next one starts on.
func netlinkAttributeBytes(kind uint16, value []byte) []byte {
	attribute := binary.NativeEndian.AppendUint16(nil, uint16(unix.SizeofRtAttr+len(value)))
	attribute = binary.NativeEndian.AppendUint16(attribute, kind)
	attribute = append(attribute, value...)

	for len(attribute)%4 != 0 {
		attribute = append(attribute, 0)
	}

	return attribute
}

// The value of a UNIX_DIAG_PEER attribute, which is an inode and nothing else.
func inodeAttributeValue(inode uint32) []byte {
	return binary.NativeEndian.AppendUint32(nil, inode)
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
