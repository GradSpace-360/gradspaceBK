package user

import (
	"fmt"
	"gradspaceBK/database"
	"gradspaceBK/middlewares"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func JobRoutes(base *fiber.Group) {
    job := base.Group("/jobs")
    job.Get("/",middlewares.AuthMiddleware , GetJobs)
    job.Get("/saved",middlewares.AuthMiddleware ,GetSavedJobs)
    job.Post("/save",middlewares.AuthMiddleware ,SaveJob)
    job.Patch("/:id/status",middlewares.AuthMiddleware,UpdateHiringStatus)
    job.Get("/my-jobs",middlewares.AuthMiddleware ,GetMyJobs)
    job.Delete("/:id",middlewares.AuthMiddleware, DeleteJob)
    job.Post("/",middlewares.AuthMiddleware, AddNewJob)
}

type JobFilters struct {
    Search     string `query:"search"`
    Location   string `query:"location"`
    CompanyID  string `query:"company_id"`
    JobType    string `query:"job_type"`
    Page       int    `query:"page"`
    Limit      int    `query:"limit"`
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
    IsSaved      bool            `json:"is_saved"`
}

type CompanyResponse struct {
    Name    string `json:"name"`
    LogoURL string `json:"logo_url"`
}

type PosterResponse struct {
    ID           string `json:"id"`
    FullName     string `json:"full_name"`
    UserName     string `json:"username"`
    ProfileImage string `json:"profile_image"`
}

type Pagination struct {
    Page  int `query:"page"`
    Limit int `query:"limit"`
}

func GetJobs(c *fiber.Ctx) error {
    user_data := c.Locals("user_data").(jwt.MapClaims)
    userID, ok := user_data["user_id"].(string)
    if !ok || userID == "" {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "success": false,
            "message": "Unauthorized: missing user ID",
        })
    }
    var filters JobFilters
    if err := c.QueryParser(&filters); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "success": false,
            "message": "Invalid query parameters",
        })
    }

    // Set defaults
    if filters.Page == 0 { filters.Page = 1 }
    if filters.Limit == 0 { filters.Limit = 10 }

    session := database.Session.Db
    var jobs []database.Job

    query := session.Model(&database.Job{}).
        Preload("Company").
        Preload("PostedByUser")

    
    if filters.Search != "" {
        query = query.Where("title ILIKE ?", "%"+filters.Search+"%")
    }
    if filters.Location != "" {
        query = query.Where("location ILIKE ?", "%"+filters.Location+"%")
    }
    if filters.CompanyID != "" {
        query = query.Where("company_id = ?", filters.CompanyID)
    }
    if filters.JobType != "" {
        query = query.Where("job_type = ?", filters.JobType)
    }
    query = query.Order("created_at DESC")
    
    var total int64
    query.Count(&total)

    offset := (filters.Page - 1) * filters.Limit
    query = query.Offset(offset).Limit(filters.Limit)

    if err := query.Find(&jobs).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "success": false,
            "message": "Failed to fetch jobs",
        })
    }
    var savedJobIDs []string
    if userID != "" {
        database.Session.Db.Model(&database.SavedJob{}).
            Where("user_id = ?", userID).
            Pluck("job_id", &savedJobIDs)
    }

    // Collect UserIDs for profile lookup
    var userIDs []string
    for _, job := range jobs {
        userIDs = append(userIDs, job.PostedByUser.ID)
    }

    // Fetch user profiles in bulk
    var userProfiles []database.UserProfile
    if len(userIDs) > 0 {
        if err := database.Session.Db.Where("user_id IN ?", userIDs).Find(&userProfiles).Error; err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "success": false,
                "message": "Failed to fetch user profiles",
            })
        }
    }

    // Create profile map
    profileMap := make(map[string]database.UserProfile)
    for _, profile := range userProfiles {
        profileMap[profile.UserID] = profile
    }

    transformedJobs := make([]JobResponse, 0, len(jobs))
    for _, job := range jobs {
        // Safe username dereference
        userName := ""
        if job.PostedByUser.UserName != nil {
            userName = *job.PostedByUser.UserName
        }

        // Get profile image
        var profileImage string
        if profile, exists := profileMap[job.PostedByUser.ID]; exists {
            profileImage = profile.ProfileImage
        }

        poster := PosterResponse{
            ID:           job.PostedByUser.ID,
            FullName:     job.PostedByUser.FullName,
            UserName:     userName,
            ProfileImage: profileImage,
        }

        transformedJobs = append(transformedJobs, JobResponse{
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
            PostedBy: poster,
            CreatedAt: job.CreatedAt.Format("2006-01-02"),
            IsSaved:      contains(savedJobIDs, job.ID),
            
        })
    }

    return c.JSON(fiber.Map{
        "success": true,
        "data": fiber.Map{
            "jobs": transformedJobs,
            "pagination": fiber.Map{
                "total": total,
                "page":  filters.Page,
                "limit": filters.Limit,
            },
        },
    })
}

func GetSavedJobs(c *fiber.Ctx) error {
	user_data := c.Locals("user_data").(jwt.MapClaims)
    userID, ok := user_data["user_id"].(string)
    if !ok || userID == "" {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "success": false,
            "message": "Unauthorized: missing user ID",
        })
    }

    var pagination struct {
        Page  int `query:"page"`
        Limit int `query:"limit"`
    }
    if err := c.QueryParser(&pagination); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "success": false,
            "message": "Invalid query parameters",
        })
    }

    if pagination.Page == 0 {
        pagination.Page = 1
    }
    if pagination.Limit == 0 {
        pagination.Limit = 10
    }
    offset := (pagination.Page - 1) * pagination.Limit

    var savedJobs []database.SavedJob
    query := database.Session.Db.
        Preload("Job.Company").
        Preload("Job.PostedByUser").
        Where("user_id = ?", userID)

    var total int64
    if err := query.Model(&database.SavedJob{}).Count(&total).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "success": false,
            "message": "Failed to count saved jobs",
        })
    }

    if err := query.Offset(offset).Limit(pagination.Limit).Find(&savedJobs).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "success": false,
            "message": "Failed to fetch saved jobs",
        })
    }

    var userIDs []string
    for _, savedJob := range savedJobs {
        userIDs = append(userIDs, savedJob.Job.PostedByUser.ID)
    }

    var userProfiles []database.UserProfile
    if len(userIDs) > 0 {
        if err := database.Session.Db.Where("user_id IN ?", userIDs).Find(&userProfiles).Error; err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "success": false,
                "message": "Failed to fetch user profiles",
            })
        }
    }

    profileMap := make(map[string]database.UserProfile)
    for _, profile := range userProfiles {
        profileMap[profile.UserID] = profile
    }

    transformedJobs := make([]JobResponse, 0, len(savedJobs))
    for _, savedJob := range savedJobs {
        job := savedJob.Job
        userName := ""
        if job.PostedByUser.UserName != nil {
            userName = *job.PostedByUser.UserName
        }

        profileImage := ""
        if profile, exists := profileMap[job.PostedByUser.ID]; exists {
            profileImage = profile.ProfileImage
        }

        transformedJob := JobResponse{
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
            PostedBy: PosterResponse{
                ID:           job.PostedByUser.ID,
                FullName:     job.PostedByUser.FullName,
                UserName:     userName,
                ProfileImage: profileImage,
            },
            IsSaved: true,
        }
        transformedJobs = append(transformedJobs, transformedJob)
    }

    return c.JSON(fiber.Map{
        "success": true,
        "data": fiber.Map{
            "jobs": transformedJobs,
            "pagination": fiber.Map{
                "total": total,
                "page":  pagination.Page,
                "limit": pagination.Limit,
            },
        },
    })
}

func GetMyJobs(c *fiber.Ctx) error {
	user_data := c.Locals("user_data").(jwt.MapClaims)
    userID, ok := user_data["user_id"].(string)
    if !ok || userID == "" {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "success": false,
            "message": "Unauthorized: missing user ID",
        })
    }
    
    var pagination struct {
        Page  int `query:"page"`
        Limit int `query:"limit"`
    }
    if err := c.QueryParser(&pagination); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "success": false,
            "message": "Invalid query parameters",
        })
    }

    if pagination.Page == 0 {
        pagination.Page = 1
    }
    if pagination.Limit == 0 {
        pagination.Limit = 10
    }
    offset := (pagination.Page - 1) * pagination.Limit

    var jobs []database.Job
    query := database.Session.Db.
        Preload("Company").
        Preload("PostedByUser").
        Where("posted_by = ?", userID)

    var total int64
    if err := query.Model(&database.Job{}).Count(&total).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "success": false,
            "message": "Failed to count jobs",
        })
    }

    if err := query.Offset(offset).Limit(pagination.Limit).Find(&jobs).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "success": false,
            "message": "Failed to fetch your jobs",
        })
    }

    var userProfile database.UserProfile
    if err := database.Session.Db.
        Where("user_id = ?", userID).
        First(&userProfile).Error; err != nil {
        userProfile = database.UserProfile{}
    }

    transformedJobs := make([]JobResponse, 0, len(jobs))
    for _, job := range jobs {
        userName := ""
        if job.PostedByUser.UserName != nil {
            userName = *job.PostedByUser.UserName
        }

        transformedJob := JobResponse{
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
            PostedBy: PosterResponse{
                ID:           userID,
                FullName:     job.PostedByUser.FullName,
                UserName:     userName,
                ProfileImage: userProfile.ProfileImage,
            },
        }
        transformedJobs = append(transformedJobs, transformedJob)
    }

    return c.JSON(fiber.Map{
        "success": true,
        "data": fiber.Map{
            "jobs": transformedJobs,
            "pagination": fiber.Map{
                "total": total,
                "page":  pagination.Page,
                "limit": pagination.Limit,
            },
        },
    })
}

func SaveJob(c *fiber.Ctx) error {
	user_data := c.Locals("user_data").(jwt.MapClaims)
    userID, ok := user_data["user_id"].(string)
    if !ok || userID == "" {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "success": false,
            "message": "Unauthorized: missing user ID",
        })
    }

    var req struct {
        JobID string `json:"job_id"`
    }

    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "success": false,
            "message": "Invalid request",
        })
    }

    // Check if already saved
    var existing database.SavedJob
    result := database.Session.Db.
        Where("user_id = ? AND job_id = ?", userID, req.JobID).
        First(&existing)

    if result.Error == nil {
        // Remove if exists
        if err := database.Session.Db.Delete(&existing).Error; err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "success": false,
                "message": "Failed to remove saved job",
            })
        }
        return c.JSON(fiber.Map{"success": true, "action": "removed"})
    }

    // Create new saved job
    newSaved := database.SavedJob{
        UserID: userID,
        JobID:  req.JobID,
    }
	fmt.Println(newSaved.UserID)
	fmt.Println(newSaved.JobID)
	// check the jobid job present in job table
	var job database.Job
	if err := database.Session.Db.Where("id = ?", req.JobID).First(&job).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid job ID",
		})
	}
    
    if err := database.Session.Db.Create(&newSaved).Error; err != nil {
		fmt.Println(err)
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "success": false,
            "message": "Failed to save job",
        })
    }

    return c.JSON(fiber.Map{"success": true, "action": "saved"})
}

func DeleteJob(c *fiber.Ctx) error {
	user_data := c.Locals("user_data").(jwt.MapClaims)
    userID, ok := user_data["user_id"].(string)
    if !ok || userID == "" {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "success": false,
            "message": "Unauthorized: missing user ID",
        })
    }
    jobID := c.Params("id")

    // Verify ownership
    result := database.Session.Db.
        Where("id = ? AND posted_by = ?", jobID, userID).
        Delete(&database.Job{})

    if result.RowsAffected == 0 {
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "success": false,
            "message": "Not authorized or job not found",
        })
    }

    return c.JSON(fiber.Map{"success": true})
}

func AddNewJob(c *fiber.Ctx) error {
    user_data := c.Locals("user_data").(jwt.MapClaims)
    userID, ok := user_data["user_id"].(string)
    if !ok || userID == "" {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "success": false,
            "message": "Unauthorized: missing user ID",
        })
    }
    
    // Define validation structure
    var req struct {
        Title        string `json:"title" validate:"required,min=5,max=255"`
        CompanyID    string `json:"company_id" validate:"required,uuid"`
        Description  string `json:"description" validate:"required,min=20"`
        Location     string `json:"location" validate:"required"`
        Requirements string `json:"requirements" validate:"required"`
        JobType      string `json:"job_type" validate:"required,oneof='Part-Time' 'Full-Time' 'Internship' 'Freelance'"`
        ApplyLink    string `json:"apply_link" validate:"omitempty,url"`
    }

    // Parse and validate request
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "success": false,
            "message": "Invalid request format",
        })
    }

    // Manual validation
    if errors := validateJobRequest(req); len(errors) > 0 {
        return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
            "success": false,
            "errors":  errors,
        })
    }

    // Verify company exists
    var company database.Company
    if err := database.Session.Db.Where("id = ?", req.CompanyID).First(&company).Error; err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "success": false,
            "message": "Invalid company ID",
        })
    }

    // Create job entity
    jobData := database.Job{
        Title:        req.Title,
        CompanyID:    req.CompanyID,
        Description:  req.Description,
        Location:     req.Location,
        Requirements: req.Requirements,
        JobType:      req.JobType,
        ApplyLink:    req.ApplyLink,
        PostedBy:     userID,
        IsOpen:       true, // Default to open status
    }

    // Save to database
    if err := database.Session.Db.Create(&jobData).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "success": false,
            "message": "Failed to create job listing",
            "error":   err.Error(),
        })
    }

    return c.Status(fiber.StatusCreated).JSON(fiber.Map{
        "success": true,
        "data": fiber.Map{
            "id":          jobData.ID,
            "title":       jobData.Title,
            "company":     company.Name,
            "location":    jobData.Location,
            "created_at":  jobData.CreatedAt,
        },
    })
}

func UpdateHiringStatus(c *fiber.Ctx) error {
	user_data := c.Locals("user_data").(jwt.MapClaims)
    userID, ok := user_data["user_id"].(string)
    if !ok || userID == "" {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "success": false,
            "message": "Unauthorized: missing user ID",
        })
    }

    jobID := c.Params("id")

    var req struct {
        IsOpen bool `json:"is_open"`
    }

    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "success": false,
            "message": "Invalid request",
        })
    }

    // Verify ownership
    var job database.Job
    if err := database.Session.Db.
        Where("id = ? AND posted_by = ?", jobID, userID).
        First(&job).Error; err != nil {
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "success": false,
            "message": "Not authorized to update this job",
        })
    }

    job.IsOpen = req.IsOpen
    if err := database.Session.Db.Save(&job).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "success": false,
            "message": "Failed to update status",
        })
    }

    return c.JSON(fiber.Map{"success": true, "is_open": job.IsOpen})
}
// Helper function for validation
func validateJobRequest(req struct {
    Title        string `json:"title" validate:"required,min=5,max=255"`
    CompanyID    string `json:"company_id" validate:"required,uuid"`
    Description  string `json:"description" validate:"required,min=20"`
    Location     string `json:"location" validate:"required"`
    Requirements string `json:"requirements" validate:"required"`
    JobType      string `json:"job_type" validate:"required,oneof='Part-Time' 'Full-Time' 'Internship' 'Freelance'"`
    ApplyLink    string `json:"apply_link" validate:"omitempty,url"`
}) map[string]string {
    errors := make(map[string]string)

    if strings.TrimSpace(req.Title) == "" {
        errors["title"] = "Title is required"
    } else if len(req.Title) < 3 {
        errors["title"] = "Title must be at least 3 characters"
    }

    if _, err := uuid.Parse(req.CompanyID); err != nil {
        errors["company_id"] = "Invalid company identifier"
    }

    if len(strings.TrimSpace(req.Description)) < 10 {
        errors["description"] = "Description must be at least 20 characters"
    }

    validJobTypes := map[string]bool{
        "Part-Time":  true,
        "Full-Time":  true,
        "Internship": true,
        "Freelance":  true,
    }
    if !validJobTypes[req.JobType] {
        errors["job_type"] = "Invalid job type specified"
    }

    if req.ApplyLink != "" {
        if _, err := url.ParseRequestURI(req.ApplyLink); err != nil {
            errors["apply_link"] = "Invalid URL format"
        }
    }

    return errors
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}