package controller

import (
	"gradspaceBK/database"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(base *fiber.Group) error {
	register := base.Group("/register")
	register.Post("/request", RegisterRequest)
	register.Patch("/:requestId", HandleRegistrationAction)
	register.Get("/requests", GetRegisterRequests)

	return nil
}

// POST /register/request
type RegisterRequestData struct {
	FullName    string `json:"full_name"`
	Department  string `json:"department"`
	Batch       string `json:"batch"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
	Role        string `json:"role"`
}

func RegisterRequest(c *fiber.Ctx) error {
	var formated_data RegisterRequestData
	if err := c.BodyParser(&formated_data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	// Add data to the register_requests table
	session := database.Session.Db
	registerRequest := database.RegisterRequest{
		FullName:    formated_data.FullName,
		Department:  formated_data.Department,
		Batch:       formated_data.Batch,
		Email:       formated_data.Email,
		PhoneNumber: formated_data.PhoneNumber,
		Role:        formated_data.Role,
	}
	session.Create(&registerRequest)
	// Update user registration_status to pending
	user := database.User{}
	if session.Model(&database.User{}).Where("email = ?", formated_data.Email).First(&user).RowsAffected > 0 {
		user.RegistrationStatus = "pending"
		session.Save(&user)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Registration request submitted successfully.",
	})
}

// PATCH /api/v1/register/{requestId}
type RegistrationAction struct {
	Action string `json:"action"`
}

func HandleRegistrationAction(c *fiber.Ctx) error {
	requestId := c.Params("requestId")
	var actionData RegistrationAction

	if err := c.BodyParser(&actionData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	session := database.Session.Db
	registerRequest := database.RegisterRequest{}
	if session.Model(&database.RegisterRequest{}).Where("id = ?", requestId).First(&registerRequest).RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Registration request not found",
		})
	}

	user := database.User{}
	if session.Model(&database.User{}).Where("email = ?", registerRequest.Email).First(&user).RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "User not found",
		})
	}

	if actionData.Action == "approve" {
		user.RegistrationStatus = "registered"
		session.Save(&user)
		session.Delete(&registerRequest)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"message": "User approved successfully.",
		})
	} else if actionData.Action == "reject" {
		user.RegistrationStatus = "rejected"
		session.Save(&user)
		session.Delete(&registerRequest)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"message": "User rejected successfully.",
		})
	}

	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"success": false,
		"message": "Invalid action",
	})
}

// GET /api/v1/register/requests
func GetRegisterRequests(c *fiber.Ctx) error {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid page number",
		})
	}

	limit, err := strconv.Atoi(c.Query("limit", "10"))
	if err != nil || limit < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid limit value",
		})
	}

	offset := (page - 1) * limit
	session := database.Session.Db

	var registerRequests []database.RegisterRequest
	var total int64
	session.Model(&database.RegisterRequest{}).Count(&total)
	session.Limit(limit).Offset(offset).Find(&registerRequests)

	data := []map[string]interface{}{}
	for _, request := range registerRequests {
		data = append(data, map[string]interface{}{
			"id":           request.ID,
			"email":        request.Email,
			"full_name":    request.FullName,
			"department":   request.Department,
			"batch":        request.Batch,
			"phone_number": request.PhoneNumber,
			"role":         request.Role,
			"created_at":   request.CreatedAt,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    data,
		"pagination": map[string]interface{}{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}
