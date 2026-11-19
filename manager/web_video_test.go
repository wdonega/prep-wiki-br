package main

import "testing"

// Exemplos tirados do library.xml de verdade: só os canais do youtube2zim são "vídeo"; o "_videos:yes" não conta.
func TestEhVideo(t *testing.T) {
	casos := []struct {
		nome   string
		livro  livroXML
		espera bool
	}{
		{"canal com autor do youtube2zim", livroXML{Criador: "Youtube Channel “Canadian Prepper”", Tags: "preppers;_videos:yes;_ftindex:no"}, true},
		{"canal só com a tag youtube", livroXML{Criador: "Alguém", Tags: "youtube;_videos:yes;_ftindex:no"}, true},
		{"wikipédia com _videos:yes", livroXML{Criador: "Wikipedia", Tags: "wikipedia;_category:wikipedia;_videos:yes"}, false},
		{"ifixit com _videos:yes", livroXML{Criador: "iFixit", Tags: "_videos:yes;iFixit;_pictures:yes"}, false},
		{"sem nada", livroXML{}, false},
	}
	for _, c := range casos {
		if got := c.livro.ehVideo(); got != c.espera {
			t.Errorf("%s: ehVideo() = %v, esperava %v", c.nome, got, c.espera)
		}
	}
}
