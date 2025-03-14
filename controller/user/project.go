package user

import (
	"encoding/json"
	"fmt"
	"gradspaceBK/database"
	"gradspaceBK/middlewares"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func ProjectRoutes(base *fiber.Group) {
	project := base.Group("/projects")
	project.Get("/", middlewares.AuthMiddleware, GetProjects)
	project.Get("/saved", middlewares.AuthMiddleware, GetSavedProjects)
	project.Post("/save", middlewares.AuthMiddleware, SaveProject)
	project.Patch("/:id/status", middlewares.AuthMiddleware, UpdateProjectStatus)
	project.Get("/my-projects", middlewares.AuthMiddleware, GetMyProjects)
	project.Delete("/:id", middlewares.AuthMiddleware, DeleteProject)
	project.Post("/", middlewares.AuthMiddleware, AddNewProject)
}

type ProjectFilters struct {
	Search       string         `query:"search"`
	ProjectType  database.ProjectType  `query:"project_type"`
	Status       database.ProjectStatus `query:"status"`
	Year         int            `query:"year"`
	Page         int            `query:"page"`
	Limit        int            `query:"limit"`
}

type ProjectResponse struct {
	ID           string                `json:"id"`
	Title        string                `json:"title"`
	Description  string                `json:"description"`
	Tags         []string              `json:"tags"`
	ProjectType  database.ProjectType  `json:"project_type"`
	Year         int                   `json:"year"`
	Mentor       string                `json:"mentor"`
	Contributors []string              `json:"contributors"`
	Links        *database.ProjectLinks `json:"links"`
	Status       database.ProjectStatus `json:"status"`
	PostedBy     PosterResponse        `json:"posted_by"`
	CreatedAt    string                `json:"created_at"`
	IsSaved      bool                  `json:"is_saved"`
}

func GetProjects(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID, _ := userData["user_id"].(string)
	var filters ProjectFilters

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

	query := session.Model(&database.Project{}).
		Preload("User").
		Order("created_at DESC")

	// Apply filters
	fmt.Println("filters.Search : ",filters.Search)
	fmt.Println("filters.Status : ",filters.Status)
	fmt.Println("filters.ProjectType : ",filters.ProjectType)
	fmt.Println("filters.Year : ",filters.Year)


	if filters.Search != "" {
		query = query.Where("title ILIKE ?", "%"+filters.Search+"%")
	}
	if filters.ProjectType != "" {
		query = query.Where("project_type = ?", filters.ProjectType)
	}
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if filters.Year > 0 {
		query = query.Where("year = ?", filters.Year)
	}

	// Pagination
	var total int64
	query.Count(&total)
	offset := (filters.Page - 1) * filters.Limit
	query = query.Offset(offset).Limit(filters.Limit)

	var projects []database.Project
	if err := query.Find(&projects).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch projects",
		})
	}
	fmt.Print("project data: ")
	for _, project := range projects {	
		fmt.Println("project : ",project)
	}
	
	// Get saved project IDs
	var savedProjectIDs []string
	if userID != "" {
		session.Model(&database.SavedProject{}).
			Where("user_id = ?", userID).
			Pluck("project_id", &savedProjectIDs)
	}

	// Collect UserIDs for profile lookup
	var userIDs []string
	for _, project := range projects {
		userIDs = append(userIDs, project.PostedBy)
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

	transformedProjects := make([]ProjectResponse, 0, len(projects))
	for _, project := range projects {
		// Get user info
		user := project.User
		userName := ""
		if user.UserName != nil {
			userName = *user.UserName
		}

		// Get profile image
		var profileImage string
		if profile, ok := profileMap[project.PostedBy]; ok {
			profileImage = profile.ProfileImage
		}

		poster := PosterResponse{
			ID:           user.ID,
			FullName:     user.FullName,
			UserName:     userName,
			ProfileImage: profileImage,
		}
			// Unmarshal JSON data
			var tags []string
			json.Unmarshal(project.Tags, &tags)
			
			var contributors []string
			json.Unmarshal(project.Contributors, &contributors)
			
			var links database.ProjectLinks
			json.Unmarshal(project.Links, &links)
			fmt.Println("links : ",links)
			fmt.Println("tags : ",tags)
			fmt.Println("contributors : ",contributors)
			transformedProjects = append(transformedProjects, ProjectResponse{
				ID:           project.ID,
				Title:        project.Title,
				Description:  project.Description,
				Tags:         tags,
				ProjectType:  project.ProjectType,
				Year:         project.Year,
				Mentor:       project.Mentor,
				Contributors: contributors,
				Links:        &links,
				Status:       project.Status,
				PostedBy:     poster,
				CreatedAt:    project.CreatedAt.Format("2006-01-02"),
				IsSaved:      contains(savedProjectIDs, project.ID),
			})
			fmt.Print("transformedProjects : ",transformedProjects)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"projects": transformedProjects,
			"pagination": fiber.Map{
				"total": total,
				"page":  filters.Page,
				"limit": filters.Limit,
			},
		},
	})
}

func GetSavedProjects(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID, _ := userData["user_id"].(string)
	var pagination struct {
		Page  int `query:"page"`
		Limit int `query:"limit"`
	}
	if err := c.QueryParser(&pagination); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid query"})
	}

	// Set defaults
	if pagination.Page == 0 { pagination.Page = 1 }
	if pagination.Limit == 0 { pagination.Limit = 10 }
	offset := (pagination.Page - 1) * pagination.Limit

	var savedProjects []database.SavedProject
	query := database.Session.Db.
		Preload("Project.User").
		Where("user_id = ?", userID)

	var total int64
	if err := query.Model(&database.SavedProject{}).Count(&total).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to count saved projects",
		})
	}

	if err := query.Offset(offset).Limit(pagination.Limit).Find(&savedProjects).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch saved projects",
		})
	}

	// Collect UserIDs for profile lookup
	var userIDs []string
	for _, saved := range savedProjects {
		userIDs = append(userIDs, saved.Project.PostedBy)
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

	transformedProjects := make([]ProjectResponse, 0, len(savedProjects))
	for _, saved := range savedProjects {
		project := saved.Project
		user := project.User
		userName := ""
		if user.UserName != nil {
			userName = *user.UserName
		}

		var profileImage string
		if profile, exists := profileMap[project.PostedBy]; exists {
			profileImage = profile.ProfileImage
		}

		poster := PosterResponse{
			ID:           user.ID,
			FullName:     user.FullName,
			UserName:     userName,
			ProfileImage: profileImage,
		}
		// Unmarshal JSON data
		var tags []string
		json.Unmarshal(project.Tags, &tags)
		
		var contributors []string
		json.Unmarshal(project.Contributors, &contributors)
		
		var links database.ProjectLinks
		json.Unmarshal(project.Links, &links)

		transformedProjects = append(transformedProjects, ProjectResponse{
			ID:           project.ID,
			Title:        project.Title,
			Description:  project.Description,
			Tags:         tags,
			ProjectType:  project.ProjectType,
			Year:         project.Year,
			Mentor:       project.Mentor,
			Contributors: contributors,
			Links:        &links,
			Status:       project.Status,
			PostedBy:     poster,
			CreatedAt:    project.CreatedAt.Format("2006-01-02"),
			IsSaved:      true,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"projects": transformedProjects,
			"pagination": fiber.Map{
				"total": total,
				"page":  pagination.Page,
				"limit": pagination.Limit,
			},
		},
	})
}

func SaveProject(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID, _ := userData["user_id"].(string)
	var req struct {
		ProjectID string `json:"project_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid request"})
	}

	// Check if already saved
	var existing database.SavedProject
	result := database.Session.Db.
		Where("user_id = ? AND project_id = ?", userID, req.ProjectID).
		First(&existing)

	if result.Error == nil {
		// Remove if exists
		if err := database.Session.Db.Delete(&existing).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Failed to remove saved project",
			})
		}
		return c.JSON(fiber.Map{"success": true, "action": "removed"})
	}

	// Verify project exists
	var project database.Project
	if err := database.Session.Db.Where("id = ?", req.ProjectID).First(&project).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid project ID",
		})
	}

	// Create new saved project
	newSaved := database.SavedProject{
		UserID:    userID,
		ProjectID: req.ProjectID,
	}
	if err := database.Session.Db.Create(&newSaved).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to save project",
		})
	}

	return c.JSON(fiber.Map{"success": true, "action": "saved"})
}

func GetMyProjects(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID, _ := userData["user_id"].(string)
	var pagination struct {
		Page  int `query:"page"`
		Limit int `query:"limit"`
	}
	if err := c.QueryParser(&pagination); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid query"})
	}

	if pagination.Page == 0 { pagination.Page = 1 }
	if pagination.Limit == 0 { pagination.Limit = 10 }
	offset := (pagination.Page - 1) * pagination.Limit

	var projects []database.Project
	query := database.Session.Db.
		Preload("User").
		Where("posted_by = ?", userID)

	var total int64
	if err := query.Model(&database.Project{}).Count(&total).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to count projects",
		})
	}

	if err := query.Offset(offset).Limit(pagination.Limit).Find(&projects).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch projects",
		})
	}

	var userProfile database.UserProfile
	if err := database.Session.Db.
		Where("user_id = ?", userID).
		First(&userProfile).Error; err != nil {
		userProfile = database.UserProfile{}
	}
	transformedProjects := make([]ProjectResponse, 0, len(projects))
	for _, project := range projects {
		// Get user info
		user := project.User
		userName := ""
		if user.UserName != nil {
			userName = *user.UserName
		}

		poster := PosterResponse{
			ID:           user.ID,
			FullName:     user.FullName,
			UserName:     userName,
			ProfileImage: userProfile.ProfileImage,
		}

			// Unmarshal JSON data
			var tags []string
			json.Unmarshal(project.Tags, &tags)
			
			var contributors []string
			json.Unmarshal(project.Contributors, &contributors)
			
			var links database.ProjectLinks
			json.Unmarshal(project.Links, &links)
	
			transformedProjects = append(transformedProjects, ProjectResponse{
				ID:           project.ID,
				Title:        project.Title,
				Description:  project.Description,
				Tags:         tags,
				ProjectType:  project.ProjectType,
				Year:         project.Year,
				Mentor:       project.Mentor,
				Contributors: contributors,
				Links:        &links,
				Status:       project.Status,
				PostedBy:     poster,
				CreatedAt:    project.CreatedAt.Format("2006-01-02"),
			})
	}


	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"projects": transformedProjects,
			"pagination": fiber.Map{
				"total": total,
				"page":  pagination.Page,
				"limit": pagination.Limit,
			},
		},
	})
}

func DeleteProject(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID, _ := userData["user_id"].(string)
	projectID := c.Params("id")

	result := database.Session.Db.
		Where("id = ? AND posted_by = ?", projectID, userID).
		Delete(&database.Project{})

	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": "Not authorized or project not found",
		})
	}
	return c.JSON(fiber.Map{"success": true})
}

func AddNewProject(c *fiber.Ctx) error {
    userData := c.Locals("user_data").(jwt.MapClaims)
    userID, _ := userData["user_id"].(string)
    
    var req struct {
        Title        string   `json:"title"`
        Description  string   `json:"description"`
        Tags         []string `json:"tags"`
        ProjectType  database.ProjectType `json:"project_type"`
        Year         int      `json:"year"`
        Mentor       string   `json:"mentor"`
        Contributors []string `json:"contributors"`
        Links        struct {
            CodeLink string `json:"code_link"`
            Video    string `json:"video"`
            Files    string `json:"files"`
            Website  string `json:"website"`
        } `json:"links"`
		Status database.ProjectStatus `json:"status"`
    }

    if err := c.BodyParser(&req); err != nil {
		fmt.Println("error : ",err)
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "success": false,
            "message": "Invalid request",
        })
    }

    // Marshal JSON data
    tagsBytes, _ := json.Marshal(req.Tags)
    contributorsBytes, _ := json.Marshal(req.Contributors)
    linksBytes, _ := json.Marshal(req.Links)

    projectData := database.Project{
        Title:        req.Title,
        Description:  req.Description,
        Tags:         tagsBytes,
        ProjectType:  req.ProjectType,
        Year:         req.Year,
        Mentor:       req.Mentor,
        Contributors: contributorsBytes,
        Links:        linksBytes,
        Status:       req.Status,
        PostedBy:     userID,
    }

    if err := database.Session.Db.Create(&projectData).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "success": false,
            "message": "Failed to create project",
        })
    }

    return c.Status(fiber.StatusCreated).JSON(fiber.Map{
        "success": true,
        "data": fiber.Map{
            "id":           projectData.ID,
            "title":        projectData.Title,
            "project_type": projectData.ProjectType,
            "status":       projectData.Status,
        },
    })
}

func UpdateProjectStatus(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID, _ := userData["user_id"].(string)
	projectID := c.Params("id")
	var req struct {
		Status database.ProjectStatus `json:"status" validate:"required,oneof=ACTIVE COMPLETED"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid request"})
	}

	// Verify ownership
	var project database.Project
	if err := database.Session.Db.
		Where("id = ? AND posted_by = ?", projectID, userID).
		First(&project).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": "Not authorized to update this project",
		})
	}

	project.Status = req.Status
	if err := database.Session.Db.Save(&project).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to update project status",
		})
	}

	return c.JSON(fiber.Map{"success": true, "status": project.Status})
}

func validateProjectRequest(req struct {
	Title        string                `json:"title" validate:"required,min=5,max=255"`
	Description  string                `json:"description" validate:"required,min=10"`
	Tags         []string              `json:"tags" validate:"required"`
	ProjectType  database.ProjectType  `json:"project_type" validate:"required,oneof=PERSONAL GROUP COLLEGE"`
	Year         int                   `json:"year" validate:"required"`
	Mentor       string                `json:"mentor"`
	Contributors []string              `json:"contributors" validate:"required"`
	Links        *database.ProjectLinks `json:"links"`
}) map[string]string {
	errors := make(map[string]string)
	
	if strings.TrimSpace(req.Title) == "" {
		errors["title"] = "Title is required"
	}
	
	if req.ProjectType == "" {
		errors["project_type"] = "Project type is required"
	}
	
	if req.Year < 2000 || req.Year > time.Now().Year()+5 {
		errors["year"] = "Invalid year value"
	}
	
	if len(req.Contributors) == 0 {
		errors["contributors"] = "At least one contributor is required"
	}
	
	if req.Links != nil {
		if req.Links.Website != "" {
			if _, err := url.ParseRequestURI(req.Links.Website); err != nil {
				errors["links.website"] = "Invalid website URL"
			}
		}
		if req.Links.CodeLink != "" {
			if _, err := url.ParseRequestURI(req.Links.CodeLink); err != nil {
				errors["links.code_link"] = "Invalid code link URL"
			}
		}
	}

	return errors
}