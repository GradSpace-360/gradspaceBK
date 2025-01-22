package controller

import (
	"github.com/gofiber/fiber/v2"

	"gradspaceBK/database"
	"gradspaceBK/util"
)

func AuthRoutes(base *fiber.Group) error {
	auth := base.Group("/auth")
	// auth.Post("/login", Login)
	auth.Post("/signup/", SignUp)
	return nil
}

type LoginData struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Login(c *fiber.Ctx) error {
	var formated_data LoginData
	var user database.User
	if err := c.BodyParser(&formated_data); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Internal Server Error",
		})
	}
	session := database.Session.Db

	email := formated_data.Email
	password := formated_data.Password

	if result := session.Where("email = ?", email).First(&user); result.Error == nil {
		if err := util.ComparePassword(password, user.Password); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Invalid Username or Password",
			})
		}
		if token, err := util.GenerateToken(user.ID); err == nil {
			access_cookie := &fiber.Cookie{
				Name:     "access_token",
				Value:    token["access_token"],
				HTTPOnly: true,
				Secure:   false,
				SameSite: "Strict",
			}
			refresh_cookie := &fiber.Cookie{
				Name:     "refresh_token",
				Value:    token["refresh_token"],
				HTTPOnly: true,
				Secure:   false,
				SameSite: "Strict",
			}
			c.Cookie(access_cookie)
			c.Cookie(refresh_cookie)
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"success": true,
				"message": "Login Successful",
				"user": map[string]interface{}{
					"username":            user.UserName,
					"full_name":           user.FullName,
					"role":                user.Role,
					"department":          user.Department,
					"batch":               user.Batch,
					"email":               user.Email,
					"is_verified":         user.IsVerified,
					"is_onboard":          user.IsOnboard,
					"registration_status": user.RegistrationStatus,
				},
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Internal Server Error",
		})

	}
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"message": "Invalid Username or Password",
	})
}

type SignUpData struct {
	UserName string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func SignUp(c *fiber.Ctx) error {
	var formated_data SignUpData

	if err := c.BodyParser(&formated_data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": err,
		})
	}
	session := database.Session.Db

	username := formated_data.UserName
	if session.Model(&database.User{}).Where("user_name = ?", username).First(&database.User{}).RowsAffected > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Username already exists",
		})
	}
	email := formated_data.Email
	user := database.User{}
	if session.Model(&database.User{}).Where("email = ?", email).First(&user).RowsAffected > 0 {
		if user.IsVerified {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Email already exists",
			})
		}
	}
	password := formated_data.Password

	hashed_password, err := util.HashPassword(password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Internal Server Error",
		})
	}

	session.Create(&database.User{Email: email, Password: hashed_password, UserName: username})
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User Created",
	})

}
