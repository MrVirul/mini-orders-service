package httpx

import "github.com/gofiber/fiber/v2"

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Error(c *fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(fiber.Map{
		"error": errorBody{Code: code, Message: message},
	})
}
