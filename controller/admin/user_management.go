package admin

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gradspaceBK/database"

	"github.com/gofiber/fiber/v2"
)

type UserFilters struct {
	Search     string `query:"search"`
	Batch      string `query:"batch"`
	Department string `query:"department"`
	Role       string `query:"role"`
	Page       int    `query:"page"`
	Limit      int    `query:"limit"`
}

type UserActionRequest struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

func AdminUserManagementRoutes(base *fiber.Group) error {
	user := base.Group("/admin/user-management")

	user.Post("/add-users/", AddUsers)
	user.Post("/promote-batch/", PromoteBatchToAlumni)
	user.Get("/users/", GetUsers)
	user.Post("/users/:id/action", PerformUserAction)

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

func GetUsers(c *fiber.Ctx) error {
	var filters UserFilters
	if err := c.QueryParser(&filters); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid query parameters",
		})
	}

	if filters.Page == 0 {
		filters.Page = 1
	}
	if filters.Limit == 0 {
		filters.Limit = 10
	}

	session := database.Session.Db
	var users []database.User
	query := session.Model(&database.User{})

	if filters.Search != "" {
		searchTerm := strings.TrimSpace(filters.Search)
		query = query.Where("full_name LIKE ? OR email LIKE ?", "%"+searchTerm+"%", "%"+searchTerm+"%")
	}
	if filters.Batch != "" {
		query = query.Where("batch = ?", filters.Batch)
	}
	if filters.Department != "" {
		query = query.Where("department = ?", filters.Department)
	}
	if filters.Role != "" {
		query = query.Where("role = ?", filters.Role)
	}

	var total int64
	query.Count(&total)

	offset := (filters.Page - 1) * filters.Limit
	query = query.Offset(offset).Limit(filters.Limit)

	if err := query.Find(&users).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch users",
		})
	}

	transformedUsers := make([]map[string]interface{}, 0, len(users))
	for _, user := range users {
		batchStr := strconv.Itoa(user.Batch)

		transformedUsers = append(transformedUsers, map[string]interface{}{
			"id":                  user.ID,
			"full_name":           user.FullName,
			"batch":               batchStr,
			"department":          user.Department,
			"email":               user.Email,
			"role":                user.Role,
			"status":              user.IsOnboard,
			"is_verified":         user.IsVerified,
			"registration_status": user.RegistrationStatus,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"users": transformedUsers,
			"pagination": fiber.Map{
				"total": total,
				"page":  filters.Page,
				"limit": filters.Limit,
			},
		},
	})
}

func PerformUserAction(c *fiber.Ctx) error {
	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "User ID is required",
		})
	}

	var actionRequest UserActionRequest
	if err := c.BodyParser(&actionRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	session := database.Session.Db
	var user database.User
	if err := session.First(&user, "id = ?", userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "User not found",
		})
	}

	switch actionRequest.Action {
	case "promote":
		if user.Role != "Student" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid action for the current user role",
			})
		}
		user.Role = "Alumni"
	case "demote":
		if user.Role != "Alumni" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid action for the current user role",
			})
		}
		user.Role = "Student"
	case "remove":
		if actionRequest.Reason == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Reason is required for this action",
			})
		}
		if err := session.Delete(&user).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Failed to remove user",
			})
		}
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid action",
		})
	}

	if err := session.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to perform action",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Action performed successfully",
	})
}