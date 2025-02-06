package controller

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"gradspaceBK/database"
	"gradspaceBK/middlewares"
)

func OnboardRoutes(base *fiber.Group) error {
	onboard := base.Group("/onboard")
	onboard.Use(middlewares.AuthMiddleware)
	onboard.Post("/complete", CompleteOnboarding)
	onboard.Post("/user-profile", CreateUserProfile)
	onboard.Post("/social-links", CreateSocialLinks)
	onboard.Post("/experience", CreateExperience)
	onboard.Post("/education", CreateEducation)

	return nil
}


func CompleteOnboarding(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID := userData["user_id"].(string)

	form, err := c.MultipartForm()
	if err != nil {
		fmt.Println("Error: Invalid form data", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid form data"})
	}

	var profileImagePath string
	if profileImage, err := c.FormFile("profile_image"); err == nil {
		uploadDir := "./uploads/profile"
		fileExt := filepath.Ext(profileImage.Filename)
		newFileName := fmt.Sprintf("%s%s", userID, fileExt)
		savePath := filepath.Join(uploadDir, newFileName)

		if err := c.SaveFile(profileImage, savePath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save image"})
		}
		profileImagePath = savePath
	}

	skills := form.Value["skills[]"]
	interests := form.Value["interests[]"]
	skillsBytes, err := json.Marshal(skills)
	if err != nil {
		fmt.Println("Error marshalling skills:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to marshal skills"})
	}
	interestsBytes, err := json.Marshal(interests)
	if err != nil {
		fmt.Println("Error marshalling interests:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to marshal interests"})
	}

	// Save User Profile
	session := database.Session.Db
	userProfile := database.UserProfile{
		UserID:       userID,
		ProfileImage: profileImagePath,
		Headline:     c.FormValue("headline"),
		About:        c.FormValue("about"),
		Location:     c.FormValue("location"),
		Skills:       skillsBytes,
		Interests:    interestsBytes,
	}
	if err := session.Create(&userProfile).Error; err != nil {
		fmt.Println("Error saving user profile:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Profile creation failed"})
	}

	var socialLinksMap map[string]string
	if socialLinksJSON, ok := form.Value["social_links"]; ok && len(socialLinksJSON) > 0 {
		if err := json.Unmarshal([]byte(socialLinksJSON[0]), &socialLinksMap); err != nil {
			fmt.Println("Error parsing social links JSON:", err)
		}
	}

	// Save Social Links
	socialLinks := database.SocialLinks{
		UserID:             userID,
		GithubURL:          socialLinksMap["github"],
		LinkedinURL:        socialLinksMap["linkedin"],
		InstagramURL:       socialLinksMap["instagram"],
		ResumeURL:          socialLinksMap["resume"],
		PersonalWebsiteURL: socialLinksMap["website"],
	}

	if err := session.Create(&socialLinks).Error; err != nil {
		fmt.Println("Error saving social links:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Social links creation failed"})
	}

	experiences := []string{}
	for key, value := range form.Value {
		if strings.HasPrefix(key, "experiences[") {
			experiences = append(experiences, value...)
		}
	}
	// Save Experiences
	if len(experiences) > 0 {
		for _, exp := range experiences {
			var expData database.Experience
			if err := json.Unmarshal([]byte(exp), &expData); err != nil {
				fmt.Println("Error parsing experience JSON:", err)
				continue
			}
			expData.UserID = userID
			if err := session.Create(&expData).Error; err != nil {
				fmt.Println("Error saving experience:", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create experience"})
			}
		}
	}

	educations := []string{}
	for key, value := range form.Value {
		if strings.HasPrefix(key, "educations[") {
			educations = append(educations, value...)
		}
	}
	// Save Educations
	if len(educations) > 0 {
		for _, edu := range educations {
			var eduData database.Education
			if err := json.Unmarshal([]byte(edu), &eduData); err != nil {
				fmt.Println("Error parsing education JSON:", err)
				continue
			}
			eduData.UserID = userID
			if err := session.Create(&eduData).Error; err != nil {
				fmt.Println("Error saving education:", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create education"})
			}
		}
	}
	// Update user's onboarding status
	if err := session.Model(&database.User{}).Where("id = ?", userID).Update("is_onboard", true).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update onboarding status"})
	}
	return c.JSON(fiber.Map{"success": true})
}



func CreateUserProfile(c *fiber.Ctx) error {
	var userProfile database.UserProfile
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID := userData["user_id"].(string)

	ProfileImage, err := c.FormFile("profile_image")
	if err == nil {
		savePath := fmt.Sprintf("./uploads/%s", ProfileImage.Filename)
		if err := c.SaveFile(ProfileImage, savePath); err != nil {
			return c.Status(500).SendString("Failed to save file")
		}
		userProfile.ProfileImage = savePath
	}
	Headline := c.FormValue("headline")
	if Headline != "" {
		userProfile.Headline = Headline
	}
	About := c.FormValue("about")
	if About != "" {
		userProfile.About = About
	}
	Location := c.FormValue("location")
	if Location != "" {
		userProfile.Location = Location
	}
	Skills := c.FormValue("skills")
	if Skills != "" {
		userProfile.Skills = []byte(Skills)
	}
	Interests := c.FormValue("interests")
	if Interests != "" {
		userProfile.Interests = []byte(Interests)
	}
	userProfile.UserID = userID

	session := database.Session.Db
	if err := session.Create(&userProfile).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to create user profile",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "User profile created successfully",
		"data":    userProfile,
	})
}

func CreateSocialLinks(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID := userData["user_id"].(string)

	type SocialLinkInput struct {
		GithubURL          *string `json:"github_url,omitempty"`
		LinkedinURL        *string `json:"linkedin_url,omitempty"`
		InstagramURL       *string `json:"instagram_url,omitempty"`
		ResumeURL          *string `json:"resume_url,omitempty"`
		PersonalWebsiteURL *string `json:"personal_website_url,omitempty"`
	}
	var formated_data SocialLinkInput
	if err := c.BodyParser(&formated_data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	socialLinks := database.SocialLinks{
		UserID:             userID,
		GithubURL:          *formated_data.GithubURL,
		LinkedinURL:        *formated_data.LinkedinURL,
		InstagramURL:       *formated_data.InstagramURL,
		ResumeURL:          *formated_data.ResumeURL,
		PersonalWebsiteURL: *formated_data.PersonalWebsiteURL,
	}
	session := database.Session.Db
	if err := session.Create(&socialLinks).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to create social links",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Social links created successfully",
		"data":    socialLinks,
	})
}

func CreateExperience(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID := userData["user_id"].(string)

	type ExperienceInput struct {
		CompanyName  string     `json:"company_name"`
		Position     string     `json:"position"`
		StartDate    time.Time  `json:"start_date"`
		EndDate      *time.Time `json:"end_date,,omitempty"`
		JobType      string     `json:"job_type"`
		LocationType string     `json:"location_type"`
		Location     string     `json:"location"`
	}
	var formated_data ExperienceInput

	if err := c.BodyParser(&formated_data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	session := database.Session.Db
	experience := database.Experience{
		UserID:       userID,
		CompanyName:  formated_data.CompanyName,
		Position:     formated_data.Position,
		StartDate:    formated_data.StartDate,
		EndDate:      formated_data.EndDate,
		JobType:      formated_data.JobType,
		LocationType: formated_data.LocationType,
		Location:     formated_data.Location,
	}
	if err := session.Create(&experience).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to create experience",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Experience created successfully",
		"data":    experience,
	})
}

func CreateEducation(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	userID := userData["user_id"].(string)

	type EducationInput struct {
		InstitutionName string     `json:"institution_name"`
		Course          string     `json:"course"`
		Location        string     `json:"location"`
		StartDate       time.Time  `json:"start_date"`
		EndDate         *time.Time `json:"end_date,omitempty"`
		Grade           string     `json:"grade"`
	}

	var formated_data EducationInput
	if err := c.BodyParser(&formated_data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
		})
	}

	session := database.Session.Db
	education := database.Education{
		UserID:          userID,
		InstitutionName: formated_data.InstitutionName,
		Course:          formated_data.Course,
		Location:        formated_data.Location,
		StartDate:       formated_data.StartDate,
		EndDate:         *formated_data.EndDate,
		Grade:           formated_data.Grade,
	}
	if err := session.Create(&education).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to create education",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Education created successfully",
		"data":    education,
	})
}
