package admin

import (
	"fmt"
	"time"

	"gradspaceBK/database"

	"github.com/gofiber/fiber/v2"
)

func AdminUserManagementRoutes(base *fiber.Group) error {
	user := base.Group("/admin/user-management")

	user.Post("/add-users/", AddUsers)
	user.Post("/promote-batch/", PromoteBatchToAlumni)

	return nil
}

func AddUsers(c *fiber.Ctx) error {
	var users []struct {
		Department string `json:"department"`
		FullName   string `json:"fullName"`
		Batch      int    `json:"batch"`
		Role       string `json:"role"`
		Email      string `json:"email"`
	}

	if err := c.BodyParser(&users); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}
	var userList []database.User
	for _, userData := range users {
		userList = append(userList, database.User{
			FullName:          userData.FullName,
			Batch:             userData.Batch,
			Department:        userData.Department,
			Role:              userData.Role,
			Email:             userData.Email,
			RegistrationStatus: "registered",
			IsVerified:        false, 
			IsOnboard:         false,
			UserName:          nil,
		})
	}
	for i, user := range userList {
		if err := validateUser(&user); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Validation error for user %d: %s", i+1, err.Error()),
			})
		}
	}
	session := database.Session.Db
	if err := session.Create(&userList).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to create users: " + err.Error(),
		})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Users added successfully",
		"users":   userList,
	})
}

func validateUser(user *database.User) error {
	if user.FullName == "" {
		return fmt.Errorf("full name is required")
	}
	if user.Batch < 1995 || user.Batch > time.Now().Year() {
		return fmt.Errorf("invalid batch year")
	}
	if user.Department == "" {
		return fmt.Errorf("department is required")
	}
	if user.Role != "Student" && user.Role != "Alumni" && user.Role != "Faculty" {
		return fmt.Errorf("invalid role")
	}
	if user.Email == "" {
		return fmt.Errorf("email is required")
	}
	return nil
}

func PromoteBatchToAlumni(c *fiber.Ctx) error {
	var request struct {
		Batch int `json:"batch"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	currentYear := time.Now().Year()
	if request.Batch < 1995 || request.Batch > currentYear-3 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": fmt.Sprintf("Batch must be between 1995 and %d", currentYear-3),
		})
	}

	session := database.Session.Db
	result := session.Model(&database.User{}).
		Where("batch = ? AND role = ?", request.Batch, "Student").
		Update("role", "Alumni")

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to promote batch: " + result.Error.Error(),
		})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("Batch %d promoted to alumni", request.Batch),
	})
}
