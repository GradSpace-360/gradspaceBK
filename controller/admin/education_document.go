// feature: based on the admin requirements parameters returning the higher education document
package admin

import (
	"strconv"

	"gradspaceBK/database"

	"github.com/gofiber/fiber/v2"
)

func EducationRoutes(base *fiber.Group) {
	admin := base.Group("/education")
	admin.Get("/higher-education", GetHigherEducation)
}


type EducationResponse struct {
    FullName         string `json:"fullName"`
    Department       string `json:"department"`
    Batch            int    `json:"batch"`
    Email            string `json:"email"`
    Course           string `json:"course"`
    InstitutionName  string `json:"institutionName"`
    Location         string `json:"location"`
    StartYear        int    `json:"startYear"`
    EndYear          *int   `json:"endYear,omitempty"`
}

func GetHigherEducation(c *fiber.Ctx) error {
    department := c.Query("department")
    batchParam := c.Query("batch")

    db := database.Session.Db

    // Build the base query
    query := db.Model(&database.Education{}).
        Select(
            "users.full_name",
            "users.department",
            "users.batch",
            "users.email",
            "educations.course",
            "educations.institution_name",
            "educations.location",
            "EXTRACT(YEAR FROM educations.start_date) AS start_year",
            "EXTRACT(YEAR FROM educations.end_date) AS end_year",
        ).
        Joins("JOIN users ON users.id = educations.user_id").
        Where("users.is_onboard = ?", true).
        Where("EXTRACT(YEAR FROM educations.start_date) >= users.batch + 3")

    // Apply optional department filter
    if department != "" {
        query = query.Where("users.department = ?", department)
    }

    // Apply optional batch filter with validation
    if batchParam != "" {
        batch, err := strconv.Atoi(batchParam)
        if err != nil {
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "Invalid batch parameter",
            })
        }
        query = query.Where("users.batch = ?", batch)
    }

    // Execute the query
    var results []EducationResponse
    if err := query.Scan(&results).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to fetch higher education data",
        })
    }

    // Post-processing to handle invalid EndYear values
    for i := range results {
        if results[i].EndYear != nil && *results[i].EndYear == 1 {
            results[i].EndYear = nil
        }
    }

    return c.JSON(results)
}