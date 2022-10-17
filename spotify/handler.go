package spotify

import "github.com/gofiber/fiber/v2"

func Handler(router fiber.Router) {
	router.Get("/playback", Playback)
	router.Get("/auth", Auth)
}