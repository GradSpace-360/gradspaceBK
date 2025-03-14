package controller

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"github.com/google/uuid"

	"gradspaceBK/database"
	"gradspaceBK/middlewares"

	"gradspaceBK/services"
	"gradspaceBK/util"
)

func AuthRoutes(base *fiber.Group) error {
	auth := base.Group("/auth")

	auth.Post("/login/", Login)
	auth.Post("/signup/", SignUp)
	auth.Get("/check-auth/", middlewares.AuthMiddleware, CheckAuth)
	auth.Get("/send-verification-otp/", middlewares.AuthMiddleware, SendVerificationOTP)
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
				UserID:            userID,
				VerificationToken: otp,
				ExpiresAt:         time.Now().Add(5 * time.Minute),
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

	// un comment this code on production environment.
	// Use email service to send the OTP
	subject := "Your Verification Code"
	text := fmt.Sprintf("Your verification code is: %s", otp)
	data := map[string]string{
		"VerificationCode": otp,
	}
	html, err := util.RenderTemplate("templates/verification_email.html", data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to render email template",
		})
	}
	if err := services.SendEmail(user.Email, subject, text, html); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to send OTP email",
		})
	}
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
	// un comment this code on production environment.
	subject := "Welcome to GradSpace!"
	data := map[string]string{
		"userName": *user.UserName,
	}
	html, err := util.RenderTemplate("templates/welcome_email.html", data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to render welcome email template",
		})
	}

	text := fmt.Sprintf("Hello %s,\n\nWelcome to GradSpace! We're thrilled to have you join our community of graduate students and researchers.", *user.UserName)
	if err := services.SendEmail(user.Email, subject, text, html); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to send welcome email",
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
		UserID:             user.ID,
		ResetPasswordToken: resetToken,
		ExpiresAt:          resetExpire,
	})

	// Generate the reset password link with the reset token
	// un comment this code on production environment.
	var resetPasswordLink string
	if os.Getenv("SERVER") == "prod" {
		resetPasswordLink = fmt.Sprintf("https://localhost:5173/reset-Password/%s", resetToken)
	} else {
		resetPasswordLink = fmt.Sprintf("https://localhost:5173/reset-Password/%s", resetToken)
	}
	data := map[string]string{
		"ResetPasswordLink": resetPasswordLink,
		"Username":          *user.UserName,
	}
	html, err := util.RenderTemplate("templates/reset_password_email.html", data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to render email template",
		})
	}

	subject := "Reset Your Password"
	text := fmt.Sprintf("Please click the following link to reset your password: %s", resetPasswordLink)
	if err := services.SendEmail(user.Email, subject, text, html); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to send reset password email",
		})
	}

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

		// Retrieve profile image from UserProfile table.
		var profile database.UserProfile
		profileResult := session.Model(&database.UserProfile{}).
			Where("user_id = ?", user.ID).
			First(&profile)
		profileImage := ""
		if profileResult.Error == nil {
			profileImage = profile.ProfileImage
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
					"username":            *user.UserName,
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
					"profile_image":       profileImage,
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
	// Check if username is already taken
	if session.Model(&database.User{}).Where("user_name = ?", username).First(&database.User{}).RowsAffected > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Username already exists",
		})
	}

	email := formated_data.Email
	var existingUser database.User
	// Check if email exists in the database
	if err := session.Where("email = ?", email).First(&existingUser).Error; err == nil {
		// Email exists
		if existingUser.IsVerified {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Email already exists",
			})
		} else {
			// Update existing unverified user with new details
			password := formated_data.Password
			hashed_password, err := util.HashPassword(password)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"message": "Internal Server Error",
				})
			}

			existingUser.UserName = &username
			existingUser.Password = hashed_password
			if err := session.Save(&existingUser).Error; err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"message": "Failed to update user",
				})
			}

			// Generate tokens and proceed
			tokens, err := util.GenerateToken(existingUser.ID)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"message": "Failed to generate tokens",
				})
			}

			// Set cookies and respond
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

			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"message": "User updated",
				"user": fiber.Map{
					"id":       existingUser.ID,
					"username": existingUser.UserName,
					"email":    existingUser.Email,
				},
			})
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Handle other database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Database error",
		})
	}

	// Email doesn't exist - create new user
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
		UserName: &username,
		// Ensure default values match admin-created users
		RegistrationStatus: "not_registered",
		IsVerified:         false,
	}

	if err := session.Create(&newUser).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to create user",
		})
	}

	// Generate tokens and respond
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

	// Retrieve the user details.
	user := database.User{}
	session.Model(&database.User{}).
		Where("id = ?", user_data["user_id"].(string)).
		First(&user)

	// Retrieve the user's profile image.
	var profile database.UserProfile
	result := session.Model(&database.UserProfile{}).
		Where("user_id = ?", user.ID).
		First(&profile)
	profileImage := ""
	if result.Error == nil {
		profileImage = profile.ProfileImage
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Authorized",
		"user": map[string]interface{}{
			"id":                  user.ID,
			"username":            *user.UserName,
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
			"profile_image":       profileImage,
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
