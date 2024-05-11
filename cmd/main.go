package main

import (
	geminiapi "github.com/guinegreiros-dev/geminiapi/gemini/api"
)

func main() {
	gemini := geminiapi.ProvideGeminiAPI()
	gemini.StartServer()
}
