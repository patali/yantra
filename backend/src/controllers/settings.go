package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/patali/yantra/src/db/models"
	"github.com/patali/yantra/src/middleware"
	"github.com/patali/yantra/src/services"
	"gorm.io/gorm"
)

type SettingsController struct {
	db *gorm.DB
}

func NewSettingsController(db *gorm.DB) *SettingsController {
	return &SettingsController{db: db}
}

// RegisterRoutes registers settings routes
func (ctrl *SettingsController) RegisterRoutes(rg *gin.RouterGroup, authService *services.AuthService) {
	settings := rg.Group("/settings")
	settings.Use(middleware.AuthMiddleware(authService))
	{
		// Node.js compatible endpoints (plural)
		settings.GET("/email-providers", ctrl.GetEmailProviders)
		settings.POST("/email-providers", ctrl.CreateEmailProvider)
		settings.PUT("/email-providers/:id", ctrl.UpdateEmailProvider)
		settings.DELETE("/email-providers/:id", ctrl.DeleteEmailProvider)
		settings.GET("/email-providers/:provider", ctrl.GetEmailProviderByName)
		settings.POST("/email-providers/test", ctrl.TestEmailProvider)
		settings.PUT("/email-providers/activate", ctrl.SetActiveEmailProvider)

		// Alternative endpoints (singular)
		settings.GET("/email", ctrl.GetEmailProviders)
		settings.POST("/email", ctrl.CreateEmailProvider)
		settings.PUT("/email/:id", ctrl.UpdateEmailProvider)
		settings.DELETE("/email/:id", ctrl.DeleteEmailProvider)
	}
}

type EmailProviderRequest struct {
	Provider        string `json:"provider" binding:"required"` // mailgun, ses, resend, smtp
	APIKey          string `json:"apiKey"`
	Domain          string `json:"domain"`
	FromEmail       string `json:"fromEmail"`
	FromName        string `json:"fromName"`
	Region          string `json:"region"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	IsActive        bool   `json:"isActive"`
	SMTPHost        string `json:"smtpHost"`
	SMTPPort        int    `json:"smtpPort"`
	SMTPUser        string `json:"smtpUser"`
	SMTPPassword    string `json:"smtpPassword"`
	SMTPSecure      *bool  `json:"smtpSecure"`
}

// GetEmailProviders returns all email providers for the account
// GET /api/settings/email-providers
func (ctrl *SettingsController) GetEmailProviders(c *gin.Context) {
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}

	var providers []models.EmailProviderSettings
	if err := ctrl.db.Where("account_id = ?", accountID).Find(&providers).Error; err != nil {
		middleware.RespondInternalError(c, err.Error())
		return
	}

	// Return empty array instead of null if no providers
	if providers == nil {
		providers = []models.EmailProviderSettings{}
	}

	middleware.RespondSuccess(c, http.StatusOK, providers)
}

// CreateEmailProvider creates or updates an email provider configuration
// POST /api/settings/email
func (ctrl *SettingsController) CreateEmailProvider(c *gin.Context) {
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}

	var req EmailProviderRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	// Check if provider already exists for this account
	var existing models.EmailProviderSettings
	result := ctrl.db.Where("account_id = ? AND provider = ?", accountID, req.Provider).First(&existing)

	if result.Error == nil {
		// Provider exists, update it
		updates := map[string]interface{}{
			"api_key":           ptrString(req.APIKey),
			"domain":            ptrString(req.Domain),
			"from_email":        ptrString(req.FromEmail),
			"from_name":         ptrString(req.FromName),
			"region":            ptrString(req.Region),
			"access_key_id":     ptrString(req.AccessKeyID),
			"secret_access_key": ptrString(req.SecretAccessKey),
			"is_active":         req.IsActive,
			"smtp_host":         ptrString(req.SMTPHost),
			"smtp_port":         ptrInt(req.SMTPPort),
			"smtp_user":         ptrString(req.SMTPUser),
			"smtp_password":     ptrString(req.SMTPPassword),
			"smtp_secure":       req.SMTPSecure,
		}

		if err := ctrl.db.Model(&existing).Updates(updates).Error; err != nil {
			middleware.RespondInternalError(c, err.Error())
			return
		}

		// Reload the updated provider
		ctrl.db.First(&existing, "id = ?", existing.ID)
		middleware.RespondSuccess(c, http.StatusOK, existing)
		return
	}

	// Provider doesn't exist, create new one
	provider := models.EmailProviderSettings{
		AccountID:       accountID,
		Provider:        req.Provider,
		APIKey:          ptrString(req.APIKey),
		Domain:          ptrString(req.Domain),
		FromEmail:       ptrString(req.FromEmail),
		FromName:        ptrString(req.FromName),
		Region:          ptrString(req.Region),
		AccessKeyID:     ptrString(req.AccessKeyID),
		SecretAccessKey: ptrString(req.SecretAccessKey),
		IsActive:        req.IsActive,
		SMTPHost:        ptrString(req.SMTPHost),
		SMTPPort:        ptrInt(req.SMTPPort),
		SMTPUser:        ptrString(req.SMTPUser),
		SMTPPassword:    ptrString(req.SMTPPassword),
		SMTPSecure:      req.SMTPSecure,
	}

	if err := ctrl.db.Create(&provider).Error; err != nil {
		middleware.RespondInternalError(c, err.Error())
		return
	}

	middleware.RespondSuccess(c, http.StatusCreated, provider)
}

// UpdateEmailProvider updates an email provider configuration
// PUT /api/settings/email/:id
func (ctrl *SettingsController) UpdateEmailProvider(c *gin.Context) {
	id := c.Param("id")
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}

	var req EmailProviderRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	// Find provider
	var provider models.EmailProviderSettings
	if err := ctrl.db.Where("id = ? AND account_id = ?", id, accountID).First(&provider).Error; err != nil {
		middleware.RespondNotFound(c, "Provider not found")
		return
	}

	// Update fields
	updates := map[string]interface{}{
		"api_key":           ptrString(req.APIKey),
		"domain":            ptrString(req.Domain),
		"from_email":        ptrString(req.FromEmail),
		"from_name":         ptrString(req.FromName),
		"region":            ptrString(req.Region),
		"access_key_id":     ptrString(req.AccessKeyID),
		"secret_access_key": ptrString(req.SecretAccessKey),
		"is_active":         req.IsActive,
		"smtp_host":         ptrString(req.SMTPHost),
		"smtp_port":         ptrInt(req.SMTPPort),
		"smtp_user":         ptrString(req.SMTPUser),
		"smtp_password":     ptrString(req.SMTPPassword),
		"smtp_secure":       req.SMTPSecure,
	}

	if err := ctrl.db.Model(&provider).Updates(updates).Error; err != nil {
		middleware.RespondInternalError(c, err.Error())
		return
	}

	// Reload
	ctrl.db.First(&provider, "id = ?", id)

	middleware.RespondSuccess(c, http.StatusOK, provider)
}

// DeleteEmailProvider deletes an email provider configuration
// DELETE /api/settings/email/:id
func (ctrl *SettingsController) DeleteEmailProvider(c *gin.Context) {
	id := c.Param("id")
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}

	result := ctrl.db.Where("id = ? AND account_id = ?", id, accountID).Delete(&models.EmailProviderSettings{})
	if result.Error != nil {
		middleware.RespondInternalError(c, result.Error.Error())
		return
	}

	if result.RowsAffected == 0 {
		middleware.RespondNotFound(c, "Provider not found")
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{"message": "Provider deleted successfully"})
}

// GetEmailProviderByName gets a specific email provider by name
// GET /api/settings/email-providers/:provider
func (ctrl *SettingsController) GetEmailProviderByName(c *gin.Context) {
	provider := c.Param("provider")
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}

	var emailProvider models.EmailProviderSettings
	if err := ctrl.db.Where("account_id = ? AND provider = ?", accountID, provider).First(&emailProvider).Error; err != nil {
		middleware.RespondNotFound(c, "Provider not found")
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, emailProvider)
}

// TestEmailProvider tests an email provider configuration
// POST /api/settings/email-providers/test
func (ctrl *SettingsController) TestEmailProvider(c *gin.Context) {
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}
	userID, err := middleware.RequireUserID(c)
	if err != nil {
		return
	}

	// Get the current user's email
	var user models.User
	if err := ctrl.db.Where("id = ?", userID).First(&user).Error; err != nil {
		middleware.RespondInternalError(c, "Failed to get user information")
		return
	}

	var req EmailProviderRequest
	if !middleware.BindJSON(c, &req) {
		return
	}

	// Create a temporary config from the request
	config := &models.EmailProviderSettings{
		Provider:        req.Provider,
		APIKey:          ptrString(req.APIKey),
		Domain:          ptrString(req.Domain),
		FromEmail:       ptrString(req.FromEmail),
		FromName:        ptrString(req.FromName),
		Region:          ptrString(req.Region),
		AccessKeyID:     ptrString(req.AccessKeyID),
		SecretAccessKey: ptrString(req.SecretAccessKey),
		SMTPHost:        ptrString(req.SMTPHost),
		SMTPPort:        ptrInt(req.SMTPPort),
		SMTPUser:        ptrString(req.SMTPUser),
		SMTPPassword:    ptrString(req.SMTPPassword),
		SMTPSecure:      req.SMTPSecure,
	}

	// Initialize email service and test the provider
	emailService := services.NewEmailService(ctrl.db)
	result, err := emailService.TestProviderToEmail(c.Request.Context(), accountID, services.EmailProvider(req.Provider), config, user.Email)

	if err != nil || !result.Success {
		errorMsg := "Failed to send test email"
		if result != nil && result.Error != "" {
			errorMsg = result.Error
		} else if err != nil {
			errorMsg = err.Error()
		}

		middleware.RespondSuccess(c, http.StatusOK, gin.H{
			"success": false,
			"error":   errorMsg,
		})
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{
		"success":   true,
		"message":   "Test email sent successfully! Check your inbox at " + user.Email,
		"messageId": result.MessageID,
	})
}

// SetActiveEmailProvider sets the active email provider
// PUT /api/settings/email-providers/activate
func (ctrl *SettingsController) SetActiveEmailProvider(c *gin.Context) {
	accountID, err := middleware.RequireAccountID(c)
	if err != nil {
		return
	}

	var req struct {
		Provider string `json:"provider" binding:"required"`
	}
	if !middleware.BindJSON(c, &req) {
		return
	}

	// Deactivate all providers for this account
	ctrl.db.Model(&models.EmailProviderSettings{}).
		Where("account_id = ?", accountID).
		Update("is_active", false)

	// Activate the specified provider
	var provider models.EmailProviderSettings
	if err := ctrl.db.Where("account_id = ? AND provider = ?", accountID, req.Provider).First(&provider).Error; err != nil {
		middleware.RespondNotFound(c, "Provider not found")
		return
	}

	provider.IsActive = true
	if err := ctrl.db.Save(&provider).Error; err != nil {
		middleware.RespondInternalError(c, err.Error())
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, provider)
}

// Helper functions
func ptrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ptrInt(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}
