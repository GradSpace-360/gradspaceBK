package admin

import (
	"gradspaceBK/database"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func RegisterAnalyticsRoutes(base *fiber.Group) error {
	analytics := base.Group("/admin/analytics")

	analytics.Get("/user-distribution", GetUserDistribution)
	analytics.Get("/department-data", GetDepartmentData)
	analytics.Get("/yearly-metrics", GetYearlyMetrics)

	return nil
}


func GetUserDistribution(c *fiber.Ctx) error {
	session := database.Session.Db

	var userCounts []struct {
		Role  string
		Count int
	}

	session.Model(&database.User{}).
		Select("role, COUNT(*) as count").
		Group("role").
		Scan(&userCounts)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    userCounts,
	})
}


func GetDepartmentData(c *fiber.Ctx) error {
	startYear, err := strconv.Atoi(c.Query("startYear"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid startYear",
		})
	}

	endYear, err := strconv.Atoi(c.Query("endYear"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid endYear",
		})
	}

	session := database.Session.Db
	var departmentStats []struct {
		Department   string
		Registered   int
		Verified     int
		Active       int
	}

	session.Raw(`
		SELECT department, 
			COUNT(*) as registered,
			SUM(CASE WHEN is_verified = true THEN 1 ELSE 0 END) as verified,
			SUM(CASE WHEN is_onboard = true THEN 1 ELSE 0 END) as active
		FROM users
		WHERE batch BETWEEN ? AND ?
		GROUP BY department
	`, startYear, endYear).Scan(&departmentStats)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    departmentStats,
	})
}


func GetYearlyMetrics(c *fiber.Ctx) error {
	startYear, err := strconv.Atoi(c.Query("startYear"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid startYear",
		})
	}

	endYear, err := strconv.Atoi(c.Query("endYear"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid endYear",
		})
	}

	session := database.Session.Db
	var yearlyStats []struct {
		Batch      int
		Registered int
		Verified   int
		Active     int
	}

	session.Raw(`
		SELECT batch, 
			COUNT(*) as registered,
			SUM(CASE WHEN is_verified = true THEN 1 ELSE 0 END) as verified,
			SUM(CASE WHEN is_onboard = true THEN 1 ELSE 0 END) as active
		FROM users
		WHERE batch BETWEEN ? AND ?
		GROUP BY batch
		ORDER BY batch ASC
	`, startYear, endYear).Scan(&yearlyStats)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    yearlyStats,
	})
}
