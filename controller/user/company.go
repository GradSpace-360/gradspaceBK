package user

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gradspaceBK/database"
	"gradspaceBK/middlewares"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func CompanyRoutes(base *fiber.Group) {
	company := base.Group("/companies")
	company.Get("/", GetCompanies)
	company.Get("/:id", GetCompany)
	company.Post("/", middlewares.AuthMiddleware, AddCompany)
	company.Put("/:id", middlewares.AuthMiddleware, UpdateCompany)
	company.Delete("/:id", middlewares.AuthMiddleware, DeleteCompany)
}

func GetCompanies(c *fiber.Ctx) error {
	var companies []database.Company
	result := database.Session.Db.Order("created_at desc").Find(&companies)

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch companies",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    companies,
	})
}

func AddCompany(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid form data",
		})
	}

	// Get form values
	name := form.Value["name"]
	if len(name) == 0 || name[0] == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Company name is required",
		})
	}

	// Handle logo upload
	logoFile, err := c.FormFile("logo")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Logo file is required",
		})
	}

	// Generate unique filename
	fileExt := filepath.Ext(logoFile.Filename)
	newFileName := fmt.Sprintf("%d-%s%s",
		time.Now().UnixNano(),
		uuid.New().String(),
		fileExt,
	)

	// Create upload directory if not exists
	uploadDir := "./uploads/company-logos"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create upload directory",
		})
	}

	// Save uploaded file
	savePath := filepath.Join(uploadDir, newFileName)
	if err := c.SaveFile(logoFile, savePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save logo file",
		})
	}

	// Create company record
	newCompany := database.Company{
		Name:    name[0],
		LogoURL: savePath,
	}

	if err := database.Session.Db.Create(&newCompany).Error; err != nil {
		// Cleanup uploaded file if DB operation fails
		os.Remove(savePath)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create company",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    newCompany,
	})
}

func UpdateCompany(c *fiber.Ctx) error {
	companyID := c.Params("id")
	var existingCompany database.Company

	// Find existing company
	if err := database.Session.Db.First(&existingCompany, "id = ?", companyID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	form, err := c.MultipartForm()
	if err == nil {
		// Handle name update
		if name, exists := form.Value["name"]; exists && len(name) > 0 {
			existingCompany.Name = name[0]
		}

		// Handle logo update
		if logoFile, err := c.FormFile("logo"); err == nil {
			// Remove old logo
			if existingCompany.LogoURL != "" {
				os.Remove(existingCompany.LogoURL)
			}

			// Generate new filename
			fileExt := filepath.Ext(logoFile.Filename)
			newFileName := fmt.Sprintf("%d-%s%s",
				time.Now().UnixNano(),
				uuid.New().String(),
				fileExt,
			)

			// Save new logo
			uploadDir := "./uploads/company-logos"
			if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to create upload directory",
				})
			}
			savePath := filepath.Join(uploadDir, newFileName)
			if err := c.SaveFile(logoFile, savePath); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to save new logo",
				})
			}
			existingCompany.LogoURL = savePath
		}
	}

	// Update company in database
	if err := database.Session.Db.Save(&existingCompany).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update company",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    existingCompany,
	})
}

func DeleteCompany(c *fiber.Ctx) error {
	companyID := c.Params("id")
	var company database.Company

	// Find and delete company
	result := database.Session.Db.Where("id = ?", companyID).First(&company)
	if result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	// Delete logo file if exists
	if company.LogoURL != "" {
		os.Remove(company.LogoURL)
	}

	// Delete company record
	if err := database.Session.Db.Delete(&company).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete company",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Company deleted successfully",
	})
}

func GetCompany(c *fiber.Ctx) error {
	companyID := c.Params("id")
	var company database.Company

	result := database.Session.Db.First(&company, "id = ?", companyID)
	if result.Error != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Company not found",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    company,
	})
}
