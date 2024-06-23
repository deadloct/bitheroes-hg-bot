package data

import _ "embed"

//go:embed help.en.template
var HelpTemplate string

//go:embed intro.en.template
var IntroTemplate string

//go:embed jokes.en.json
var JokesJSON []byte

//go:embed phrases.en.json
var PhrasesJSON []byte
