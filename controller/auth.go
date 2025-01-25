package controller

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

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
	auth.Get("/send-verification-otp/", middlewares.AuthMiddleware,SendVerificationOTP)
	auth.Post("/verify-email", middlewares.AuthMiddleware, VerifyEmail) 
	auth.Post("/logout/", Logout)
	auth.Post("/forgot-password", ForgotPassword) 
	auth.Post("/reset-password/:token", ResetPassword)
	return nil
}

func SendVerificationOTP(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID := userData["user_id"].(string)

	session := database.Session.Db

	var user database.User
	if err := session.Where("id = ?", userID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "User not found",
		})
	}

	if user.IsVerified {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "User is already verified",
		})
	}

	otp, err := util.GenerateOtp()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to generate OTP",
		})
	}

	var verification database.Verification

	if err := session.Where("user_id = ?", userID).First(&verification).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			verification = database.Verification{
				UserID:               userID,
				VerificationToken:    otp,
				ExpiresAt: time.Now().Add(5 * time.Minute),
			}
			if err := session.Create(&verification).Error; err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"message": "Failed to store OTP",
				})
			}
		} else {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Failed to query verification record",
			})
		}
	} else {
		verification.VerificationToken = otp
		verification.ExpiresAt = time.Now().Add(5 * time.Minute)
		if err := session.Save(&verification).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Failed to update OTP",
			})
		}
	}

	// TODO: Implement email sending service to send OTP
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "OTP sent successfully",
	})
}

func VerifyEmail(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID := userData["user_id"].(string)

	type VerifyEmailRequest struct {
		Code string `json:"code"`
	}

	var request VerifyEmailRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
		})
	}

	session := database.Session.Db

	var user database.User
	if err := session.First(&user, "id = ?", userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "User not found",
		})
	}

	if user.IsVerified {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "User is already verified",
		})
	}

	var verification database.Verification
	if err := session.First(&verification, "user_id = ?", userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "OTP not found or expired",
		})
	}

	if time.Now().After(verification.ExpiresAt) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "OTP has expired",
		})
	}

	if request.Code != verification.VerificationToken {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid OTP",
		})
	}

	if err := session.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&database.User{}).Where("id = ?", userID).Update("is_verified", true).Error; err != nil {
			return err
		}
		if err := tx.Delete(&verification).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to verify user",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Email verified successfully",
	})
}


func ForgotPassword(c *fiber.Ctx) error {
	type RequestBody struct {
		Email string `json:"email"`
	}

	var body RequestBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
		})
	}

	session := database.Session.Db
	user := database.User{}
	if session.Where("email = ?", body.Email).First(&user).RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "User not found",
		})
	}

	resetToken := uuid.New().String()
	resetExpire := time.Now().Add(5 * time.Minute)

	session.Create(&database.Verification{
		UserID:           user.ID,
		ResetPasswordToken: resetToken,
		ExpiresAt: resetExpire,
	})

	// TODO: Send email with resetToken to user.Email

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Reset password email sent",
	})
}
func ResetPassword(c *fiber.Ctx) error {
	type ResetPasswordRequest struct {
		Password string `json:"password"`
	}

	token := c.Params("token")

	var request ResetPasswordRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
		})
	}
	
	session := database.Session.Db
	var verification database.Verification
	if err := session.Where("reset_password_token = ?", token).First(&verification).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Invalid or expired token",
		})
	}

	if time.Now().After(verification.ExpiresAt) {
		session.Delete(&verification)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Token has expired. Please request a new password reset.",
		})
	}

	hashedPassword, err := util.HashPassword(request.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to hash password",
		})
	}

	if err := session.Model(&database.User{}).Where("id = ?", verification.UserID).Update("password", hashedPassword).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to update password",
		})
	}

	if err := session.Delete(&verification).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to delete verification record",
		})
	}
	// TODO: Send email to user.Email informing them that their password has been reset
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Password has been reset successfully",
	})
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
			"message": err.Error(),
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

	newUser := database.User{
		Email:    email,
		Password: hashed_password,
		UserName: username,
	}
	if err := session.Create(&newUser).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to create user",
		})
	}

	tokens, err := util.GenerateToken(newUser.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to generate tokens",
		})
	}

	access_cookie := &fiber.Cookie{
		Name:     "access_token",
		Value:    tokens["access_token"],
		HTTPOnly: true,
		Secure:   false,
		SameSite: "None",
	}
	refresh_cookie := &fiber.Cookie{
		Name:     "refresh_token",
		Value:    tokens["refresh_token"],
		HTTPOnly: true,
		Secure:   false,
		SameSite: "None",
	}
	c.Cookie(access_cookie)
	c.Cookie(refresh_cookie)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User Created",
		"user": fiber.Map{
			"id":       newUser.ID,
			"username": newUser.UserName,
			"email":    newUser.Email,
		},
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
