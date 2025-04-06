package admin

import (
	"gradspaceBK/database"
	"math"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

type JobReportSummary struct {
	JobID       string    `json:"job_id"`
	JobType     string    `json:"job_type"`
	Title       string    `json:"title"`
	PostedBy    string    `json:"-"`
	UserFullName string   `json:"-"`
	FlagCount   int       `json:"flag_count"`
	TopReason   string    `json:"top_reason"`
	LastFlagged time.Time `json:"-"`
	PostDate    time.Time `json:"-"`
}

type JobResponse struct {
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	Location     string          `json:"location"`
	Requirements string          `json:"requirements"`
	IsOpen       bool            `json:"is_open"`
	JobType      string          `json:"job_type"`
	ApplyLink    string          `json:"apply_link"`
	Company      CompanyResponse `json:"company"`
	PostedBy     PosterResponse  `json:"posted_by"`
	CreatedAt    string          `json:"created_at"`
}


type CompanyResponse struct {
	Name    string `json:"name"`
	LogoURL string `json:"logo_url"`
}

func AdminJobRoutes(base *fiber.Group) {
	admin := base.Group("/admin")
	admin.Get("/job-reports", GetJobReports)
	admin.Get("/job-reports/:jobId", GetJobReportDetails)
	admin.Delete("/job-reports/:jobId/dismiss", DismissJobReports)
	admin.Delete("/jobs/:jobId", AdminDeleteJob)
}

func GetJobReports(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	if page < 1 { page = 1 }
	if limit < 1 || limit > 100 { limit = 10 }
	offset := (page - 1) * limit

	var summaries []JobReportSummary
	query := `
		SELECT 
			j.id AS job_id,
			j.job_type AS job_type,
			j.title AS title,
			j.posted_by AS posted_by,
			u.full_name AS user_full_name,
			COUNT(jr.id) AS flag_count,
			(SELECT reason 
			 FROM job_reports 
			 WHERE job_id = j.id 
			 GROUP BY reason 
			 ORDER BY COUNT(*) DESC 
			 LIMIT 1) AS top_reason,
			MAX(jr.created_at) AS last_flagged,
			j.created_at AS post_date
		FROM job_reports jr
		JOIN jobs j ON jr.job_id = j.id
		JOIN users u ON j.posted_by = u.id
		GROUP BY j.id, u.full_name, j.created_at
		ORDER BY last_flagged DESC
		LIMIT ? OFFSET ?
	`

	if err := database.Session.Db.Raw(query, limit, offset).Scan(&summaries).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch job reports",
		})
	}

	response := make([]fiber.Map, 0)
	for _, s := range summaries {
		response = append(response, fiber.Map{
			"job_id":       s.JobID,
			"job_type":     s.JobType,
			"title":        s.Title,
			"posted_by":    fiber.Map{"id": s.PostedBy, "name": s.UserFullName},
			"flag_count":   s.FlagCount,
			"top_reason":   s.TopReason,
			"last_flagged": s.LastFlagged.Format(time.RFC3339),
			"post_date":   s.PostDate.Format("2006-01-02"),
		})
	}

	var total int64
	database.Session.Db.Model(&database.JobReport{}).
		Select("COUNT(DISTINCT job_id)").
		Scan(&total)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    response,
		"meta": fiber.Map{
			"current_page": page,
			"per_page":     limit,
			"total_pages":  int(math.Ceil(float64(total) / float64(limit))),
			"total_items":  total,
		},
	})
}

func GetJobReportDetails(c *fiber.Ctx) error {
	jobID := c.Params("jobId")

	var job database.Job
	if err := database.Session.Db.
		Preload("PostedByUser").
		Preload("Company").
		Where("id = ?", jobID).
		First(&job).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Job not found",
		})
	}

	var reports []database.JobReport
	if err := database.Session.Db.
		Preload("Reporter").
		Where("job_id = ?", jobID).
		Find(&reports).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch reports",
		})
	}

	// Fetch user profile for poster
	var userProfile database.UserProfile
	database.Session.Db.Where("user_id = ?", job.PostedByUser.ID).First(&userProfile)

	userName := ""
	if job.PostedByUser.UserName != nil {
		userName = *job.PostedByUser.UserName
	}
	// Build poster response
	if job.PostedByUser.UserName != nil {
		userName = *job.PostedByUser.UserName
	}
	poster := PosterResponse{
		ID:           job.PostedByUser.ID,
		FullName:     job.PostedByUser.FullName,
		UserName:     userName,
		ProfileImage: userProfile.ProfileImage, // Assuming profile image exists in User model
	}

	jobResponse := JobResponse{
		ID:           job.ID,
		Title:        job.Title,
		Description:  job.Description,
		Location:     job.Location,
		Requirements: job.Requirements,
		IsOpen:       job.IsOpen,
		JobType:      job.JobType,
		ApplyLink:    job.ApplyLink,
		Company: CompanyResponse{
			Name:    job.Company.Name,
			LogoURL: job.Company.LogoURL,
		},
		PostedBy:  poster,
		CreatedAt: job.CreatedAt.Format("2006-01-02"),
	}

	flagData := make([]fiber.Map, 0)
	for _, r := range reports {
		flagData = append(flagData, fiber.Map{
			"reporter": fiber.Map{
				"id":   r.ReporterID,
				"name": r.Reporter.FullName,
			},
			"reason":    r.Reason,
			"flag_date": r.CreatedAt.Format(time.RFC3339),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"job_data":  jobResponse,
			"flag_data": flagData,
		},
	})
}

func DismissJobReports(c *fiber.Ctx) error {
	jobID := c.Params("jobId")
	if jobID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid job ID",
		})
	}

	var count int64
	if err := database.Session.Db.Model(&database.JobReport{}).
		Where("job_id = ?", jobID).
		Count(&count).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to check reports",
		})
	}

	if count == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "No reports found",
		})
	}

	tx := database.Session.Db.Begin()
	if err := tx.Where("job_id = ?", jobID).Delete(&database.JobReport{}).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to dismiss reports",
		})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Commit failed",
		})
	}

	return c.JSON(fiber.Map{
		"success":         true,
		"message":         "Reports dismissed",
		"dismissed_count": count,
	})
}

func AdminDeleteJob(c *fiber.Ctx) error {
	jobID := c.Params("jobId")
	if jobID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid job ID",
		})
	}

	tx := database.Session.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Delete job
	if err := tx.Where("id = ?", jobID).Delete(&database.Job{}).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to delete job",
		})
	}

	// Cleanup reports
	if err := tx.Where("job_id = ?", jobID).Delete(&database.JobReport{}).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to cleanup reports",
		})
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Commit failed",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Job deleted successfully",
	})
}