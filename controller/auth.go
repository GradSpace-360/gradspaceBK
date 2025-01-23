package controller

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"gradspaceBK/database"
	"gradspaceBK/middlewares"
	"gradspaceBK/util"
)

func AuthRoutes(base *fiber.Group) error {
	auth := base.Group("/auth")

	auth.Post("/login/", Login)
	auth.Post("/signup/", SignUp)
	auth.Get("/check-auth/", middlewares.AuthMiddleware, CheckAuth)
	auth.Get("/send-verification/", middlewares.AuthMiddleware, SendVerification)
	auth.Get("/verify/:token", middlewares.AuthMiddleware, VerifyUser)
	auth.Post("/logout/", Logout)
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

	if result := session.Model(&database.User{}).Where("email = ?", email).First(&user); result.Error == nil {
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
				SameSite: "None",
			}
			refresh_cookie := &fiber.Cookie{
				Name:     "refresh_token",
				Value:    token["refresh_token"],
				HTTPOnly: true,
				Secure:   false,
				SameSite: "None",
			}
			c.Cookie(access_cookie)
			c.Cookie(refresh_cookie)
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"success": true,
				"message": "Login Successful",
				"user": map[string]interface{}{
					"id":                  user.ID,
					"username":            user.UserName,
					"full_name":           user.FullName,
					"role":                user.Role,
					"department":          user.Department,
					"batch":               user.Batch,
					"email":               user.Email,
					"is_verified":         user.IsVerified,
					"is_onboard":          user.IsOnboard,
					"registration_status": user.RegistrationStatus,
					"created_at":          user.CreatedAt,
					"updated_at":          user.UpdatedAt,
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

func CheckAuth(c *fiber.Ctx) error {
	user_data := c.Locals("user_data").(jwt.MapClaims)
	session := database.Session.Db
	user := database.User{}
	session.Model(&database.User{}).Where("id = ?", user_data["user_id"].(string)).First(&user)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Authorized",
		"user": map[string]interface{}{
			"id":                  user.ID,
			"username":            user.UserName,
			"full_name":           user.FullName,
			"role":                user.Role,
			"department":          user.Department,
			"batch":               user.Batch,
			"email":               user.Email,
			"is_verified":         user.IsVerified,
			"is_onboard":          user.IsOnboard,
			"registration_status": user.RegistrationStatus,
			"created_at":          user.CreatedAt,
			"updated_at":          user.UpdatedAt,
		},
	})
}

func SendVerification(c *fiber.Ctx) error {
	user_data := c.Locals("user_data").(jwt.MapClaims)
	session := database.Session.Db
	user := database.User{}
	if session.Model(&database.User{}).Where("id = ?", user_data["user_id"].(string)).First(&user).RowsAffected == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "User not found",
		})
	}
	if user.IsVerified {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "User is already verified",
		})
	}
	token := uuid.New().String()
	session.Create(&database.Verification{UserID: user.ID, VerificationToken: token})
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Verification email sent",
	})
}

func VerifyUser(c *fiber.Ctx) error {
	user_data := c.Locals("user_data").(jwt.MapClaims)
	verificationToken := c.Params("token")
	session := database.Session.Db
	verification := database.Verification{}
	if session.Model(&database.Verification{}).Where(
		"user_id = ? and verification_token = ?", user_data["user_id"], verificationToken).First(&verification).RowsAffected == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid Token",
		})
	}
	if verification.CreatedAt.Add(5 * time.Minute).Before(time.Now()) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Token Expired",
		})
	}
	session.Model(&database.User{}).Where("id = ?", user_data["user_id"]).Update("is_verified", true)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "User Verified",
	})
}

func Logout(c *fiber.Ctx) error {
	// Create cookies with empty values and expired dates to clear them
	access_cookie := &fiber.Cookie{
		Name:     "access_token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		Secure:   false,
		SameSite: "None",
	}
	refresh_cookie := &fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		Secure:   false,
		SameSite: "None",
	}
	c.Cookie(access_cookie)
	c.Cookie(refresh_cookie)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Logout successful",
	})
}
