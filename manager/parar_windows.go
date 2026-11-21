//go:build windows

package main

import "os"

// interromper pede a um processo filho (kiwix-serve, llama-server) que pare. O Windows não tem sinal de
// interrupção para processos sem console (os.Interrupt devolve erro e o processo seguiria vivo), então
// aqui ele é encerrado de vez. Os dois só leem arquivos: nada fica pela metade.
func interromper(p *os.Process) error { return p.Kill() }
