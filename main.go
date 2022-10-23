package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/nanderLP/api/spotify"
	"github.com/nanderLP/api/websocket"
	"log"
)

func main() {
	// load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	spotifyGroup := app.Group("/spotify")
	spotify.Handler(spotifyGroup)

	websocketGroup := app.Group("/ws")
	websocket.Handler(websocketGroup)

	app.Listen(":3000")
}
