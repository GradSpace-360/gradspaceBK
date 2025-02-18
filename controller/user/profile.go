package user

import (
	"encoding/json"
	"fmt"
	"gradspaceBK/database"
	"gradspaceBK/middlewares"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func RegisterProfileRoutes(base *fiber.Group) error {
    profile := base.Group("profile") // removed trailing slash for consistency
    profile.Patch("/profileImage",middlewares.AuthMiddleware,UpdateProfileImage)
    profile.Get("/:userName", GetUserProfile)
    profile.Patch("/:userName",middlewares.AuthMiddleware,UpdateUserProfile)
    return nil
}

func GetUserProfile(c *fiber.Ctx) error {
    userNameParam := c.Params("userName")
    session := database.Session.Db
    
    user := database.User{}
    if err := session.First(&user, "user_name = ?", userNameParam).Error; err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
    }

    var userProfile database.UserProfile
    if err := session.Where("user_id = ?", user.ID).First(&userProfile).Error; err != nil {
        if err != gorm.ErrRecordNotFound {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error fetching user profile"})
        }
    }

    var skills []string
    if len(userProfile.Skills) > 0 {
        if err := json.Unmarshal(userProfile.Skills, &skills); err != nil {
            log.Printf("Error unmarshaling skills for user %s: %v", user.ID, err)
        }
    }

    var interests []string
    if len(userProfile.Interests) > 0 {
        if err := json.Unmarshal(userProfile.Interests, &interests); err != nil {
            log.Printf("Error unmarshaling interests for user %s: %v", user.ID, err)
        }
    }

    var socialLinks database.SocialLinks
    if err := session.Where("user_id = ?", user.ID).First(&socialLinks).Error; err != nil {
        if err != gorm.ErrRecordNotFound {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error fetching social links"})
        }
    }
    
    var experiences []database.Experience
    if err := session.Where("user_id = ?", user.ID).Find(&experiences).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error fetching experiences"})
    }

    formattedExperiences := make([]fiber.Map, len(experiences))
    for i, exp := range experiences {
        endDate := ""
        if exp.EndDate != nil {
            endDate = exp.EndDate.Format(time.RFC3339)
        }
        formattedExperiences[i] = fiber.Map{
            "companyName":  exp.CompanyName,
            "position":     exp.Position,
            "startDate":    exp.StartDate.Format(time.RFC3339),
            "endDate":      endDate,
            "jobType":      exp.JobType,
            "locationType": exp.LocationType,
            "location":     exp.Location,
        }
    }

    var educations []database.Education
    if err := session.Where("user_id = ?", user.ID).Find(&educations).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error fetching educations"})
    }

    formattedEducations := make([]fiber.Map, len(educations))
    for i, edu := range educations {
        endDate := ""
        if !edu.EndDate.IsZero() {
            endDate = edu.EndDate.Format(time.RFC3339)
        }
        formattedEducations[i] = fiber.Map{
            "institutionName": edu.InstitutionName,
            "course":          edu.Course,
            "location":        edu.Location,
            "startDate":       edu.StartDate.Format(time.RFC3339),
            "endDate":         endDate,
            "grade":           edu.Grade,
        }
    }

    username := ""
    if user.UserName != nil {
        username = *user.UserName
    }
    // TODO: Add POSTS FIELD ,SO THAT WE CAN FETCH POSTS OF USER
    // TODO: Add FOLLOWERS FIELD ,SO THAT WE CAN FETCH FOLLOWERS OF USER
    // TODO: Add FOLLOWING FIELD ,SO THAT WE CAN FETCH FOLLOWING OF USER
    
    response := fiber.Map{
        "user": fiber.Map{
            "id":         user.ID,
            "fullName":   user.FullName,
            "userName":   username,
            "department": user.Department,
            "batch":      user.Batch,
            "role":       user.Role,
        },
        "profile": fiber.Map{
            "profileImage": userProfile.ProfileImage,
            "headline":     userProfile.Headline,
            "about":        userProfile.About,
            "location":     userProfile.Location,
            "skills":       skills,
            "interests":    interests,
        },
        "socialLinks": fiber.Map{
            "github":    socialLinks.GithubURL,
            "linkedin":  socialLinks.LinkedinURL,
            "instagram": socialLinks.InstagramURL,
            "resume":    socialLinks.ResumeURL,
            "website":   socialLinks.PersonalWebsiteURL,
        },
        "experiences": formattedExperiences,
        "educations":  formattedEducations,
    }
    return c.JSON(response)
}

func UpdateUserProfile(c *fiber.Ctx) error {
    userData := c.Locals("user_data").(jwt.MapClaims)
    userID := userData["user_id"].(string)
    session := database.Session.Db

  
    form, err := c.MultipartForm()
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid form data"})
    }

 
    var requestData struct {
        Profile      struct {
            Headline  string   `json:"headline"`
            About     string   `json:"about"`
            Location  string   `json:"location"`
            Skills    []string `json:"skills"`
            Interests []string `json:"interests"`
        } `json:"profile"`
        SocialLinks struct {
            Github    string `json:"github"`
            Linkedin  string `json:"linkedin"`
            Instagram string `json:"instagram"`
            Resume    string `json:"resume"`
            Website   string `json:"website"`
        } `json:"socialLinks"`
        Experiences []struct {
            CompanyName  string `json:"companyName"`
            Position     string `json:"position"`
            StartDate    string `json:"startDate"`
            EndDate      string `json:"endDate"`
            JobType      string `json:"jobType"`
            LocationType string `json:"locationType"`
            Location     string `json:"location"`
        } `json:"experiences"`
        Educations []struct {
            InstitutionName string `json:"institutionName"`
            Course          string `json:"course"`
            Location        string `json:"location"`
            StartDate       string `json:"startDate"`
            EndDate         string `json:"endDate"`
            Grade           string `json:"grade"`
        } `json:"educations"`
    }

    dataValues := form.Value["data"]
    if len(dataValues) == 0 {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Missing data field"})
    }

    if err := json.Unmarshal([]byte(dataValues[0]), &requestData); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON data"})
    }

    tx := session.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    var existingProfile database.UserProfile
    if err := tx.Where("user_id = ?", userID).First(&existingProfile).Error; err == nil {
        // Update profile fields
        existingProfile.Headline = requestData.Profile.Headline
        existingProfile.About = requestData.Profile.About
        existingProfile.Location = requestData.Profile.Location

        skills, _ := json.Marshal(requestData.Profile.Skills)
        interests, _ := json.Marshal(requestData.Profile.Interests)
        existingProfile.Skills = skills
        existingProfile.Interests = interests

        if err := tx.Save(&existingProfile).Error; err != nil {
            tx.Rollback()
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update profile"})
        }
    } else {
        // Create new profile if none exists,probably this will never be executed,
        // we are always create profile when user is onboarded.
        skills, _ := json.Marshal(requestData.Profile.Skills)
        interests, _ := json.Marshal(requestData.Profile.Interests)
        newProfile := database.UserProfile{
            UserID:    userID,
            Headline:  requestData.Profile.Headline,
            About:     requestData.Profile.About,
            Location:  requestData.Profile.Location,
            Skills:    skills,
            Interests: interests,
        }
        if err := tx.Create(&newProfile).Error; err != nil {
            tx.Rollback()
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create profile"})
        }
    }

    if err := tx.Where("user_id = ?", userID).Delete(&database.SocialLinks{}).Error; err != nil {
        tx.Rollback()
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to clear social links"})
    }

    newSocial := database.SocialLinks{
        UserID:             userID,
        GithubURL:          requestData.SocialLinks.Github,
        LinkedinURL:        requestData.SocialLinks.Linkedin,
        InstagramURL:       requestData.SocialLinks.Instagram,
        ResumeURL:          requestData.SocialLinks.Resume,
        PersonalWebsiteURL: requestData.SocialLinks.Website,
    }
    if err := tx.Create(&newSocial).Error; err != nil {
        tx.Rollback()
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save social links"})
    }

    if err := tx.Where("user_id = ?", userID).Delete(&database.Experience{}).Error; err != nil {
        tx.Rollback()
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to clear experiences"})
    }

    for _, exp := range requestData.Experiences {
        startDate, err := time.Parse(time.RFC3339, exp.StartDate)
        if err != nil {
            tx.Rollback()
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid experience start date"})
        }

        var endDate *time.Time
        if exp.EndDate != "" {
            ed, err := time.Parse(time.RFC3339, exp.EndDate)
            if err != nil {
                tx.Rollback()
                return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid experience end date"})
            }
            endDate = &ed
        }

        experience := database.Experience{
            UserID:       userID,
            CompanyName:  exp.CompanyName,
            Position:     exp.Position,
            StartDate:    startDate,
            EndDate:      endDate,
            JobType:      exp.JobType,
            LocationType: exp.LocationType,
            Location:     exp.Location,
        }

        if err := tx.Create(&experience).Error; err != nil {
            tx.Rollback()
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save experience"})
        }
    }

    if err := tx.Where("user_id = ?", userID).Delete(&database.Education{}).Error; err != nil {
        tx.Rollback()
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to clear educations"})
    }

    for _, edu := range requestData.Educations {
        startDate, err := time.Parse(time.RFC3339, edu.StartDate)
        if err != nil {
            tx.Rollback()
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid education start date"})
        }

        endDate, err := time.Parse(time.RFC3339, edu.EndDate)
        if err != nil {
            tx.Rollback()
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid education end date"})
        }

        education := database.Education{
            UserID:          userID,
            InstitutionName: edu.InstitutionName,
            Course:          edu.Course,
            Location:        edu.Location,
            StartDate:       startDate,
            EndDate:         endDate,
            Grade:           edu.Grade,
        }

        if err := tx.Create(&education).Error; err != nil {
            tx.Rollback()
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save education"})
        }
    }

    if err := tx.Commit().Error; err != nil {
        tx.Rollback()
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Transaction failed"})
    }

    return c.JSON(fiber.Map{"message": "Profile updated successfully"})
}

func UpdateProfileImage(c *fiber.Ctx) error {
    // may be weired the way we are handling the profileimage,can be optimized, but for now it works

    // Ensure multipart form is parsed even if no file is attached.
    _, _ = c.MultipartForm()

    userData := c.Locals("user_data").(jwt.MapClaims)
    userID := userData["user_id"].(string)
    session := database.Session.Db

    newProfileImagePath := ""

    // Attempt to get a file from the form first.
    profileImageFile, fileErr := c.FormFile("profile_image")
    if fileErr == nil {
        // New binary image provided
        uploadDir := "./uploads/profile"
        fileExt := filepath.Ext(profileImageFile.Filename)
        newFileName := fmt.Sprintf("%s%s", userID, fileExt)
        savePath := filepath.Join(uploadDir, newFileName)

        if err := c.SaveFile(profileImageFile, savePath); err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save image"})
        }
        newProfileImagePath = savePath
    } else {
        // No file was attached; check the form value for "profile_image"
        profileImageField := c.FormValue("profile_image")
        if profileImageField == "null" {
            // Remove existing image from disk if exists
            var userProfile database.UserProfile
            if err := session.Where("user_id = ?", userID).First(&userProfile).Error; err == nil {
                if userProfile.ProfileImage != "" {
                    _ = os.Remove(userProfile.ProfileImage)
                }
            }
            newProfileImagePath = ""
        } else if profileImageField != "" {
            // No change: retain the existing image path
            var userProfile database.UserProfile
            if err := session.Where("user_id = ?", userID).First(&userProfile).Error; err == nil {
                newProfileImagePath = userProfile.ProfileImage
            }
        } else {
            // If no file and no valid form field, then return an error
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid form data"})
        }
    }

    // Update the user profile record
    var userProfile database.UserProfile
    if err := session.Where("user_id = ?", userID).First(&userProfile).Error; err == nil {
        userProfile.ProfileImage = newProfileImagePath
        if err := session.Save(&userProfile).Error; err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update profile image"})
        }
    } else {
        // Create a new profile record if not found
        newProfile := database.UserProfile{
            UserID:       userID,
            ProfileImage: newProfileImagePath,
        }
        if err := session.Create(&newProfile).Error; err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save profile image"})
        }
    }

    return c.JSON(fiber.Map{
        "message":      "Profile image updated successfully",
        "profileImage": newProfileImagePath,
    })
}