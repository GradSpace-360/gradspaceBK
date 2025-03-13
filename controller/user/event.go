package user

import (
	"fmt"
	"gradspaceBK/database"
	"gradspaceBK/middlewares"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func EventRoutes(base *fiber.Group) {
	event := base.Group("/events")
	event.Get("/", middlewares.AuthMiddleware, GetEvents)
	event.Get("/saved", middlewares.AuthMiddleware, GetSavedEvents)
	event.Post("/save", middlewares.AuthMiddleware, SaveEvent)
	event.Patch("/:id/status", middlewares.AuthMiddleware, UpdateRegistrationStatus)
	event.Get("/my-events", middlewares.AuthMiddleware, GetMyEvents)
	event.Delete("/:id", middlewares.AuthMiddleware, DeleteEvent)
	event.Post("/", middlewares.AuthMiddleware, AddNewEvent)
}

type EventFilters struct {
	Search    string `query:"search"`
	Venue     string `query:"venue"`
	EventType string `query:"event_type"`
	// expects date in "2006-01-02" format; if not selected, this will be an empty string
	StartDate string `query:"start_date"`
	Page      int    `query:"page"`
	Limit     int    `query:"limit"`
}

type EventResponse struct {
	ID                 string         `json:"id"`
	Title              string         `json:"title"`
	Description        string         `json:"description"`
	Venue              string         `json:"venue"`
	EventType          string         `json:"event_type"`
	RegisterLink       string         `json:"register_link"`
	StartDateTime      time.Time      `json:"start_date_time"`
	EndDateTime        time.Time      `json:"end_date_time"`
	IsRegistrationOpen bool           `json:"is_registration_open"`
	// PosterResponse is defined in the jobs.go file under the user package
	PostedBy           PosterResponse `json:"posted_by"`
	CreatedAt          string         `json:"created_at"`
	IsSaved            bool           `json:"is_saved"`
}

func GetEvents(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID, _ := userData["user_id"].(string)
	var filters EventFilters
	queryString := c.Context().QueryArgs().String()
	fmt.Println("Debug: URL Query Parameters:", queryString)
	if err := c.QueryParser(&filters); err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid query parameters",
		})
	}

	// Set defaults
	if filters.Page == 0 { filters.Page = 1 }
	if filters.Limit == 0 { filters.Limit = 10 }

	session := database.Session.Db

	query := session.Model(&database.Event{}).
		Preload("User").
		Order("start_date_time DESC")

	// Apply filters
	if filters.Search != "" {
		query = query.Where("title ILIKE ?", "%"+filters.Search+"%")
	}
	if filters.Venue != "" {
		query = query.Where("venue ILIKE ?", "%"+filters.Venue+"%")
	}
	if filters.EventType != "" {
		query = query.Where("event_type = ?", filters.EventType)
	}
	if filters.StartDate != "" {
		startDate, err := time.Parse("2006-01-02", filters.StartDate)
		if err == nil {
			query = query.Where("start_date_time >= ?", startDate)
		}
	}

	// Pagination
	var total int64
	query.Count(&total)
	offset := (filters.Page - 1) * filters.Limit
	query = query.Offset(offset).Limit(filters.Limit)

	var events []database.Event

	if err := query.Find(&events).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to fetch events",
		})
	}

	// Get saved event IDs
	var savedEventIDs []string
	if userID != "" {
		session.Model(&database.SavedEvent{}).
			Where("user_id = ?", userID).
			Pluck("event_id", &savedEventIDs)
	}

    // Collect UserIDs for profile lookup
    var userIDs []string
    for _, event := range events {
        userIDs = append(userIDs, event.User.ID)
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


	transformedEvents := make([]EventResponse, 0, len(events))
	for _, event := range events {
		userName := ""
		if event.User.UserName != nil {
			userName = *event.User.UserName
		}

		// Get profile image
		var profileImage string
		if profile, ok := profileMap[event.User.ID]; ok {
			profileImage = profile.ProfileImage
		}

		poster := PosterResponse{
			ID:           event.User.ID,
			FullName:     event.User.FullName,
			UserName:     userName,
			ProfileImage: profileImage,
		}


		transformedEvents = append(transformedEvents, EventResponse{
			ID:                 event.ID,
			Title:              event.Title,
			Description:        event.Description,
			Venue:              event.Venue,
			EventType:          string(event.EventType),
			RegisterLink:       event.RegisterLink,
			StartDateTime:      event.StartDateTime,
			EndDateTime:        event.EndDateTime,
			IsRegistrationOpen: event.IsRegistrationOpen,
			PostedBy: poster,
			CreatedAt: event.CreatedAt.Format("2006-01-02"),
			// containes is a helper function defined in the jobs.go file under the user package
			IsSaved:   contains(savedEventIDs, event.ID),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"events": transformedEvents,
			"pagination": fiber.Map{
				"total": total,
				"page":  filters.Page,
				"limit": filters.Limit,
			},
		},
	})
}

func GetSavedEvents(c *fiber.Ctx) error {
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
	
	var savedEvents []database.SavedEvent  
	query := database.Session.Db.  
		Preload("Event.User").  
		Where("user_id = ?", userID)  
	
	var total int64  
	if err := query.Model(&database.SavedEvent{}).Count(&total).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "success": false,
            "message": "Failed to count saved events",
        })
    }
	
	if err := query.Offset(offset).Limit(pagination.Limit).Find(&savedEvents).Error; err != nil {  
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{  
			"success": false,  
			"message": "Failed to fetch saved events",  
		})  
	}  


	// Collect UserIDs for profile lookup
	var userIDs []string
	for _, savedEvent := range savedEvents {
		userIDs = append(userIDs, savedEvent.Event.PostedBy)
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

	
	transformedEvents := make([]EventResponse, 0, len(savedEvents))  
	for _, savedEvent := range savedEvents {  
		event := savedEvent.Event  
		userName := ""  
		if event.User.UserName != nil {  
			userName = *event.User.UserName  
		}  
		profileImage := ""
        if profile, exists := profileMap[event.User.ID]; exists {
            profileImage = profile.ProfileImage
        }

		poster := PosterResponse{
			ID:           event.User.ID,
			FullName:     event.User.FullName,
			UserName:     userName,
			ProfileImage: profileImage,
		}

		transformedEvents = append(transformedEvents, EventResponse{  
			ID:                 event.ID,  
			Title:              event.Title,  
			Description:        event.Description,  
			Venue:              event.Venue,  
			EventType:          string(event.EventType),  
			RegisterLink:       event.RegisterLink,  
			StartDateTime:      event.StartDateTime,  
			EndDateTime:        event.EndDateTime,  
			IsRegistrationOpen: event.IsRegistrationOpen,  
			PostedBy: poster,
			IsSaved: true,  
		})  
	}  
	
	return c.JSON(fiber.Map{  
		"success": true,  
		"data": fiber.Map{  
			"events": transformedEvents,  
			"pagination": fiber.Map{  
				"total": total,  
				"page":  pagination.Page,  
				"limit": pagination.Limit,  
			},  
		},  
	})  
}

func GetMyEvents(c *fiber.Ctx) error {
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
	
	var events []database.Event  
	query := database.Session.Db.  
		Preload("User").  
		Where("posted_by = ?", userID)  
	
	var total int64  
	if err := query.Model(&database.Event{}).Count(&total).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "success": false,
            "message": "Failed to count events",
        })
    }
	
	if err := query.Offset(offset).Limit(pagination.Limit).Find(&events).Error; err != nil {  
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{  
			"success": false,  
			"message": "Failed to fetch your events",  
		})  
	}  

	var userProfile database.UserProfile
    if err := database.Session.Db.
        Where("user_id = ?", userID).
        First(&userProfile).Error; err != nil {
        userProfile = database.UserProfile{}
    }
	
	transformedEvents := make([]EventResponse, 0, len(events))  
	for _, event := range events {  
		userName := ""  
		if event.User.UserName != nil {  
			userName = *event.User.UserName  
		}  
	
		transformedEvents = append(transformedEvents, EventResponse{  
			ID:                 event.ID,  
			Title:              event.Title,  
			Description:        event.Description,  
			Venue:              event.Venue,  
			EventType:          string(event.EventType),  
			RegisterLink:       event.RegisterLink,  
			StartDateTime:      event.StartDateTime,  
			EndDateTime:        event.EndDateTime,  
			IsRegistrationOpen: event.IsRegistrationOpen,  
			PostedBy: PosterResponse{  
				ID:           event.User.ID,  
				FullName:     event.User.FullName,  
				UserName:     userName,  
				ProfileImage: userProfile.ProfileImage,  
			},  
			CreatedAt: event.CreatedAt.Format("2006-01-02"),  
		})  
	}  
	
	return c.JSON(fiber.Map{  
		"success": true,  
		"data": fiber.Map{  
			"events": transformedEvents,  
			"pagination": fiber.Map{  
				"total": total,  
				"page":  pagination.Page,  
				"limit": pagination.Limit,  
			},  
		},  
	})  
}

func SaveEvent(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID, _ := userData["user_id"].(string)
	var req struct {  
		EventID string `json:"event_id"`  
	}  
	if err := c.BodyParser(&req); err != nil {  
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid request"})  
	}  
	
	// Check if already saved  
	var existing database.SavedEvent  
	result := database.Session.Db.  
		Where("user_id = ? AND event_id = ?", userID, req.EventID).  
		First(&existing)  
	
	if result.Error == nil {  
		// Remove if exists  
		if err := database.Session.Db.Delete(&existing).Error; err != nil {  
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{  
				"success": false,  
				"message": "Failed to remove saved event",  
			})  
		}  
		return c.JSON(fiber.Map{"success": true, "action": "removed"})  
	}  
	
	// Verify event exists  
	var event database.Event  
	if err := database.Session.Db.Where("id = ?", req.EventID).First(&event).Error; err != nil {  
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{  
			"success": false,  
			"message": "Invalid event ID",  
		})  
	}  
	
	// Create new saved event  
	newSaved := database.SavedEvent{  
		UserID:  userID,  
		EventID: req.EventID,  
	}  
	if err := database.Session.Db.Create(&newSaved).Error; err != nil {  
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{  
			"success": false,  
			"message": "Failed to save event",  
		})  
	}  
	
	return c.JSON(fiber.Map{"success": true, "action": "saved"})  
}

func DeleteEvent(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID, _ := userData["user_id"].(string)
	eventID := c.Params("id")
	// Verify ownership
	result := database.Session.Db.
		Where("id = ? AND posted_by = ?", eventID, userID).
		Delete(&database.Event{})

	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"message": "Not authorized or event not found",
		})
	}
	return c.JSON(fiber.Map{"success": true})
}

func AddNewEvent(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID, _ := userData["user_id"].(string)
	var req struct {  
		Title              string    `json:"title" validate:"required,min=5,max=255"`  
		Description        string    `json:"description" validate:"required,min=10"`  
		Venue              string    `json:"venue" validate:"required"`  
		EventType          string    `json:"event_type" validate:"required,oneof=CAMPUS_EVENT ALUM_EVENT"`  
		RegisterLink       string    `json:"register_link" validate:"omitempty,url"`  
		StartDateTime      time.Time `json:"start_date_time" validate:"required"`  
		EndDateTime        time.Time `json:"end_date_time" validate:"required,gtfield=StartDateTime"`  
	}  
	
	if err := c.BodyParser(&req); err != nil {  
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid request"})  
	}  
	
	// Manual validation  
	errors := validateEventRequest(req)  
	if len(errors) > 0 {  
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{  
			"success": false,  
			"errors":  errors,  
		})  
	}  
	
	// Create event  
	eventData := database.Event{  
		Title:              req.Title,  
		Description:        req.Description,  
		Venue:              req.Venue,  
		EventType:          database.EventType(req.EventType),  
		RegisterLink:       req.RegisterLink,  
		StartDateTime:      req.StartDateTime,  
		EndDateTime:        req.EndDateTime,  
		IsRegistrationOpen: true,  
		PostedBy:           userID,  
	}  
	
	if err := database.Session.Db.Create(&eventData).Error; err != nil {  
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{  
			"success": false,  
			"message": "Failed to create event",  
		})  
	}  
	
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{  
		"success": true,  
		"data": fiber.Map{  
			"id":          eventData.ID,  
			"title":       eventData.Title,  
			"venue":       eventData.Venue,  
			"event_type":  eventData.EventType,  
			"start_time":  eventData.StartDateTime,  
		},  
	})  
}


func UpdateRegistrationStatus(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID, _ := userData["user_id"].(string)
	eventID := c.Params("id")
	var req struct {  
		IsOpen bool `json:"is_open"`  
	}  
	if err := c.BodyParser(&req); err != nil {  
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "Invalid request"})  
	}  
	
	// Verify ownership  
	var event database.Event  
	if err := database.Session.Db.  
		Where("id = ? AND posted_by = ?", eventID, userID).  
		First(&event).Error; err != nil {  
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{  
			"success": false,  
			"message": "Not authorized to update this event",  
		})  
	}  
	
	event.IsRegistrationOpen = req.IsOpen  
	if err := database.Session.Db.Save(&event).Error; err != nil {  
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{  
			"success": false,  
			"message": "Failed to update registration status",  
		})  
	}  
	
	return c.JSON(fiber.Map{"success": true, "is_open": event.IsRegistrationOpen})  
}

// Helper functions
func validateEventRequest(req struct {
		Title              string    `json:"title" validate:"required,min=5,max=255"`  
		Description        string    `json:"description" validate:"required,min=10"`  
		Venue              string    `json:"venue" validate:"required"`  
		EventType          string    `json:"event_type" validate:"required,oneof=CAMPUS_EVENT ALUM_EVENT"`  
		RegisterLink       string    `json:"register_link" validate:"omitempty,url"`  
		StartDateTime      time.Time `json:"start_date_time" validate:"required"`  
		EndDateTime        time.Time `json:"end_date_time" validate:"required,gtfield=StartDateTime"`  
	}) map[string]string {
	errors := make(map[string]string)
	if strings.TrimSpace(req.Title) == "" {  
		errors["title"] = "Title is required"  
	}  
	
	if req.EventType != "CAMPUS_EVENT" && req.EventType != "ALUM_EVENT" {  
		errors["event_type"] = "Invalid event type"  
	}  
	
	if req.StartDateTime.After(req.EndDateTime) {  
		errors["end_date_time"] = "End time must be after start time"  
	}  
	
	if req.RegisterLink != "" {  
		if _, err := url.ParseRequestURI(req.RegisterLink); err != nil {  
			errors["register_link"] = "Invalid URL format"  
		}  
	}  
	
	return errors  
}



