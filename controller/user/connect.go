// package user

// import (
// 	"encoding/json"
// 	"fmt"
// 	"strings"

// 	"gradspaceBK/database"
// 	"gradspaceBK/middlewares"

// 	"github.com/gofiber/fiber/v2"
// 	"github.com/golang-jwt/jwt/v5"
// 	"gorm.io/gorm"
// )

// func ConnectRoutes(base *fiber.Group) {
// 	connect := base.Group("/connect")
// 	connect.Get("/suggested-users", middlewares.AuthMiddleware, GetSuggestedConnectUsers)
// 	connect.Get("/users", middlewares.AuthMiddleware, GetUsers)
// }

// func GetSuggestedConnectUsers(c *fiber.Ctx) error {
// 	userData := c.Locals("user_data").(jwt.MapClaims)
// 	currentUserID := userData["user_id"].(string)
// 	session := database.Session.Db

// 	var currentUserProfile database.UserProfile
// 	if err := session.Where("user_id = ?", currentUserID).First(&currentUserProfile).Error; err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch profile"})
// 	}

// 	var currentSkills, currentInterests []string
// 	json.Unmarshal(currentUserProfile.Skills, &currentSkills)
// 	json.Unmarshal(currentUserProfile.Interests, &currentInterests)

// 	query := session.Model(&database.User{}).
// 		Select("users.*").
// 		Joins("JOIN user_profiles ON user_profiles.user_id = users.id").
// 		Where("users.is_verified = ? AND users.is_onboard = ?", true, true).
// 		Where("users.id != ? AND users.id NOT IN (?)", currentUserID, session.Model(&database.Follow{}).
// 			Select("following_id").Where("follower_id = ?", currentUserID))

// 	var conditions []string
// 	var params []interface{}
// 	for _, s := range currentSkills {
// 		conditions = append(conditions, "user_profiles.skills::text ILIKE ?")
// 		params = append(params, "%"+s+"%")
// 	}
// 	for _, i := range currentInterests {
// 		conditions = append(conditions, "user_profiles.interests::text ILIKE ?")
// 		params = append(params, "%"+i+"%")
// 	}
// 	if len(conditions) > 0 {
// 		query = query.Where(strings.Join(conditions, " OR "), params...)
// 	}

// 	var suggestedUsers []database.User
// 	if err := query.Limit(10).Find(&suggestedUsers).Error; err != nil {
// 		fmt.Println("err:", err)
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch users"})
// 	}

// 	return buildUserResponse(c, suggestedUsers, currentUserID)
// }

// type UserFilters struct {
// 	Department string   `query:"department"`
// 	Role       string   `query:"role"`
// 	Batch      int      `query:"batch"`
// 	Skills     []string `query:"skills"`
// 	Interests  []string `query:"interests"`
// 	Search     string   `query:"search"`
// 	Page       int      `query:"page"`
// 	Limit      int      `query:"limit"`
// }

// func GetUsers(c *fiber.Ctx) error {
// 	userData := c.Locals("user_data").(jwt.MapClaims)
// 	currentUserID := userData["user_id"].(string)
// 	session := database.Session.Db

// 	var filters UserFilters
// 	if err := c.QueryParser(&filters); err != nil {
// 		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid query"})
// 	}

// 	if filters.Page == 0 {
// 		filters.Page = 1
// 	}
// 	if filters.Limit == 0 {
// 		filters.Limit = 10
// 	}

// 	query := session.Model(&database.User{}).
// 		Joins("JOIN user_profiles ON user_profiles.user_id = users.id").
// 		Where("users.is_verified = ? AND users.is_onboard = ?", true, true)

// 	if filters.Department != "" {
// 		query = query.Where("users.department = ?", filters.Department)
// 	}
// 	if filters.Role != "" {
// 		query = query.Where("users.role = ?", filters.Role)
// 	}
// 	if filters.Batch != 0 {
// 		query = query.Where("users.batch = ?", filters.Batch)
// 	}
// 	if len(filters.Skills) > 0 {
// 		query = addLikeConditions(query, "user_profiles.skills::text", filters.Skills)
// 	}
// 	if len(filters.Interests) > 0 {
// 		query = addLikeConditions(query, "user_profiles.interests::text", filters.Interests)
// 	}
// 	if filters.Search != "" {
// 		search := "%" + filters.Search + "%"
// 		query = query.Where("users.full_name ILIKE ? OR users.user_name ILIKE ?", search, search)
// 	}

// 	var total int64
// 	query.Count(&total)
// 	query = query.Offset((filters.Page - 1) * filters.Limit).Limit(filters.Limit)

// 	var users []database.User
// 	if err := query.Find(&users).Error; err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch users"})
// 	}

// 	return buildUserResponse(c, users, currentUserID)
// }

// func addLikeConditions(query *gorm.DB, column string, terms []string) *gorm.DB {
// 	var conditions []string
// 	var params []interface{}
// 	for _, term := range terms {
// 		conditions = append(conditions, column+" ILIKE ?")
// 		params = append(params, "%"+term+"%")
// 	}
// 	return query.Where(strings.Join(conditions, " OR "), params...)
// }

// func buildUserResponse(c *fiber.Ctx, users []database.User, currentUserID string) error {
// 	userIDs := make([]string, len(users))
// 	for i, u := range users {
// 		userIDs[i] = u.ID
// 	}

// 	var profiles []database.UserProfile
// 	database.Session.Db.Where("user_id IN ?", userIDs).Find(&profiles)
// 	profileMap := make(map[string]database.UserProfile)
// 	for _, p := range profiles {
// 		profileMap[p.UserID] = p
// 	}

// 	var followedIDs []string
// 	database.Session.Db.Model(&database.Follow{}).
// 		Where("follower_id = ? AND following_id IN ?", currentUserID, userIDs).
// 		Pluck("following_id", &followedIDs)
// 	followedMap := make(map[string]bool)
// 	for _, id := range followedIDs {
// 		followedMap[id] = true
// 	}

// 	response := make([]fiber.Map, len(users))
// 	for i, user := range users {
// 		profile := profileMap[user.ID]
// 		username := ""
// 		if user.UserName != nil {
// 			username = *user.UserName
// 		}

// 		response[i] = fiber.Map{
// 			"id":           user.ID,
// 			"fullName":     user.FullName,
// 			"userName":     username,
// 			"profileImage": profile.ProfileImage,
// 			"department":   user.Department,
// 			"batch":        user.Batch,
// 			"role":         user.Role,
// 			"isFollowing":  followedMap[user.ID],
// 		}
// 	}

// 	return c.JSON(fiber.Map{"users": response})
// }

package user

import (
	"encoding/json"
	"fmt"
	"strings"

	"gradspaceBK/database"
	"gradspaceBK/middlewares"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func ConnectRoutes(base *fiber.Group) {
	connect := base.Group("/connect")
	connect.Get("/suggested-users", middlewares.AuthMiddleware, GetSuggestedConnectUsers)
	connect.Get("/users", middlewares.AuthMiddleware, GetUsers)
}

func GetSuggestedConnectUsers(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	currentUserID := userData["user_id"].(string)
	session := database.Session.Db

	var currentUserProfile database.UserProfile
	if err := session.Where("user_id = ?", currentUserID).First(&currentUserProfile).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Failed to fetch profile"})
	}

	var currentSkills, currentInterests []string
	json.Unmarshal(currentUserProfile.Skills, &currentSkills)
	json.Unmarshal(currentUserProfile.Interests, &currentInterests)

	query := session.Model(&database.User{}).
		Select("users.*").
		Joins("JOIN user_profiles ON user_profiles.user_id = users.id").
		Where("users.is_verified = ? AND users.is_onboard = ?", true, true).
		Where("users.id != ? AND users.id NOT IN (?)", currentUserID, session.Model(&database.Follow{}).
			Select("following_id").Where("follower_id = ?", currentUserID))

	var conditions []string
	var params []interface{}
	for _, s := range currentSkills {
		conditions = append(conditions, "user_profiles.skills::text ILIKE ?")
		params = append(params, "%"+s+"%")
	}
	for _, i := range currentInterests {
		conditions = append(conditions, "user_profiles.interests::text ILIKE ?")
		params = append(params, "%"+i+"%")
	}
	if len(conditions) > 0 {
		query = query.Where(strings.Join(conditions, " OR "), params...)
	}

	var suggestedUsers []database.User
	if err := query.Limit(10).Find(&suggestedUsers).Error; err != nil {
		fmt.Println("err:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Failed to fetch users"})
	}

	response, err := buildUserResponse(suggestedUsers, currentUserID)
	if err != nil {
		fmt.Println("err building response:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Failed to build user response"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"users": response,
		},
	})
}

type UserFilters struct {
	Department string   `query:"department"`
	Role       string   `query:"role"`
	Batch      int      `query:"batch"`
	Skills     []string `query:"skills"`
	Interests  []string `query:"interests"`
	Search     string   `query:"search"`
	Page       int      `query:"page"`
	Limit      int      `query:"limit"`
}

func GetUsers(c *fiber.Ctx) error {
	userData := c.Locals("user_data").(jwt.MapClaims)
	currentUserID := userData["user_id"].(string)
	session := database.Session.Db

	var filters UserFilters
	if err := c.QueryParser(&filters); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "Invalid query parameters"})
	}

	if filters.Page == 0 {
		filters.Page = 1
	}
	if filters.Limit == 0 {
		filters.Limit = 10
	}

	query := session.Model(&database.User{}).
		Joins("JOIN user_profiles ON user_profiles.user_id = users.id").
		Where("users.is_verified = ? AND users.is_onboard = ?", true, true)

	if filters.Department != "" {
		query = query.Where("users.department = ?", filters.Department)
	}
	if filters.Role != "" {
		query = query.Where("users.role = ?", filters.Role)
	}
	if filters.Batch != 0 {
		query = query.Where("users.batch = ?", filters.Batch)
	}
	if len(filters.Skills) > 0 {
		query = addLikeConditions(query, "user_profiles.skills::text", filters.Skills)
	}
	if len(filters.Interests) > 0 {
		query = addLikeConditions(query, "user_profiles.interests::text", filters.Interests)
	}
	if filters.Search != "" {
		search := "%" + filters.Search + "%"
		query = query.Where("users.full_name ILIKE ? OR users.user_name ILIKE ?", search, search)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Failed to count users"})
	}

	query = query.Offset((filters.Page - 1) * filters.Limit).Limit(filters.Limit)

	var users []database.User
	if err := query.Find(&users).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Failed to fetch users"})
	}

	response, err := buildUserResponse(users, currentUserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Failed to build user response"})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"users": response,
			"pagination": fiber.Map{
				"total": total,
				"page":  filters.Page,
				"limit": filters.Limit,
			},
		},
	})
}

func addLikeConditions(query *gorm.DB, column string, terms []string) *gorm.DB {
	var conditions []string
	var params []interface{}
	for _, term := range terms {
		conditions = append(conditions, column+" ILIKE ?")
		params = append(params, "%"+term+"%")
	}
	return query.Where(strings.Join(conditions, " OR "), params...)
}

func buildUserResponse(users []database.User, currentUserID string) ([]fiber.Map, error) {
	userIDs := make([]string, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}

	var profiles []database.UserProfile
	if err := database.Session.Db.Where("user_id IN ?", userIDs).Find(&profiles).Error; err != nil {
		return nil, err
	}

	profileMap := make(map[string]database.UserProfile)
	for _, p := range profiles {
		profileMap[p.UserID] = p
	}

	var followedIDs []string
	if err := database.Session.Db.Model(&database.Follow{}).
		Where("follower_id = ? AND following_id IN ?", currentUserID, userIDs).
		Pluck("following_id", &followedIDs).Error; err != nil {
		return nil, err
	}

	followedMap := make(map[string]bool)
	for _, id := range followedIDs {
		followedMap[id] = true
	}

	response := make([]fiber.Map, len(users))
	for i, user := range users {
		profile := profileMap[user.ID]
		username := ""
		if user.UserName != nil {
			username = *user.UserName
		}

		response[i] = fiber.Map{
			"id":           user.ID,
			"fullName":     user.FullName,
			"userName":     username,
			"profileImage": profile.ProfileImage,
			"department":   user.Department,
			"batch":        user.Batch,
			"role":         user.Role,
			"isFollowing":  followedMap[user.ID],
		}
	}

	return response, nil
}