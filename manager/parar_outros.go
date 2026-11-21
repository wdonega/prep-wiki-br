//go:build !windows

package main

import "os"

// interromper pede a um processo filho (kiwix-serve, llama-server) que pare, com tempo para fechar direito.
func interromper(p *os.Process) error { return p.Signal(os.Interrupt) }
