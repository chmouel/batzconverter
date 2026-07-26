//go:build unix

package app

import (
	"testing"

	"github.com/creack/pty"
)

func TestIsTerminalPTY(t *testing.T) {
	terminal, peer, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()
	defer peer.Close()
	if !isTerminal(terminal) {
		t.Fatal("PTY was not detected as a terminal")
	}
}
