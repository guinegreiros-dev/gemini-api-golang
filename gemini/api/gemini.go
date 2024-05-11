package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/caarlos0/env/v6"
	"github.com/google/generative-ai-go/genai"
	"github.com/gorilla/mux"
	"google.golang.org/api/option"
)

type config struct {
	ApiKey   string `env:"API_KEY,required,notEmpty"`
	HttpPort string `env:"HTTP_PORT,notEmpty" envDefault:"8085"`
}

type geminiAPI struct {
	client genai.Client
	config
}

func ProvideGeminiAPI() geminiAPI {
	var config config
	if err := env.Parse(&config); err != nil {
		panic(err)
	}
	ctx := context.Background()

	// Access your API key as an environment variable (see "Set up your API key" above)
	client, err := genai.NewClient(ctx, option.WithAPIKey(config.ApiKey))
	if err != nil {
		log.Fatal(err)
	}

	return geminiAPI{
		client: *client,
		config: config,
	}
}

func (g *geminiAPI) StartServer() {
	router := mux.NewRouter()

	router.HandleFunc("/", g.generateText).Methods("POST")

	fmt.Print("Running server on port:", g.config.HttpPort)

	log.Fatal(http.ListenAndServe(":"+g.config.HttpPort, router))
}

func (g *geminiAPI) generateText(w http.ResponseWriter, r *http.Request) {
	typeValue := r.FormValue("type")
	textValue := r.FormValue("text")
	ctx := context.Background()

	if typeValue == "" ||
		textValue == "" {
		http.Error(w, "missing mandatory fields. type or text", http.StatusBadRequest)
		return
	}

	//Getting image
	image, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var imageBytes bytes.Buffer
	_, errReadImage := io.Copy(&imageBytes, image)
	if errReadImage != nil {
		http.Error(w, "error reading image", http.StatusBadRequest)
		return
	}

	//Based in type, modal or multimodal, determinates the type of generation for text
	switch typeValue {
	case "modal":
		res, err := g.generateTextWithText(ctx, textValue)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(res)
	case "multimodal":
		res, err := g.generateTextWithTextAndImage(ctx, textValue, imageBytes.Bytes())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(res)
	default:
		http.Error(w, "invalid type. Select modal or multimodal", http.StatusBadRequest)
	}
}

func (g *geminiAPI) generateTextWithText(ctx context.Context, text string) ([]genai.Part, error) {
	model := g.client.GenerativeModel("gemini-pro")
	resp, err := model.GenerateContent(ctx, genai.Text(text))
	if err != nil {
		return nil, fmt.Errorf("error generating content to text: %s. error: %w", text, err)
	}

	for _, candidate := range resp.Candidates {

		return candidate.Content.Parts, nil
	}

	return nil, fmt.Errorf("error generating response")
}

func (g *geminiAPI) generateTextWithTextAndImage(ctx context.Context, text string, image []byte) ([]genai.Part, error) {
	// For text-and-image input (multimodal), use the gemini-pro-vision model
	model := g.client.GenerativeModel("gemini-pro-vision")

	prompt := []genai.Part{
		genai.ImageData("png", image),
		genai.Text(text),
	}
	resp, err := model.GenerateContent(ctx, prompt...)
	if err != nil {
		return nil, fmt.Errorf("error generating content to text: %s, with this image, error: %w", text, err)
	}

	for _, candidate := range resp.Candidates {

		return candidate.Content.Parts, nil
	}

	return nil, fmt.Errorf("error generating response")
}
