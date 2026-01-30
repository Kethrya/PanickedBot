package commands

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"PanickedBot/internal/db"
	"PanickedBot/internal/discord"
)

// cleanCSVContent removes markdown code blocks and blank lines from CSV content
func cleanCSVContent(content string) string {
	lines := strings.Split(content, "\n")
	var cleaned []string
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Skip blank lines
		if trimmed == "" {
			continue
		}
		
		// Skip markdown code block markers
		if trimmed == "```" || trimmed == "```csv" || strings.HasPrefix(trimmed, "```") {
			continue
		}
		
		cleaned = append(cleaned, trimmed)
	}
	
	return strings.Join(cleaned, "\n")
}

// parseWarCSV parses a CSV file with war data
// First line: date in DD-MM-YY format
// Remaining lines: family_name, kills, deaths
func parseWarCSV(content io.Reader) (warDate time.Time, warLines []db.WarLineData, err error) {
	reader := csv.NewReader(content)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // Allow variable number of fields per record

	// Read first line (date)
	dateRecord, err := reader.Read()
	if err != nil {
		return time.Time{}, nil, fmt.Errorf("failed to read date line: %w", err)
	}

	if len(dateRecord) == 0 {
		return time.Time{}, nil, fmt.Errorf("date line is empty")
	}

	// Parse date
	warDate, err = time.Parse("02-01-06", strings.TrimSpace(dateRecord[0]))
	if err != nil {
		return time.Time{}, nil, fmt.Errorf("invalid date format (expected DD-MM-YY): %w", err)
	}

	// Read remaining lines (war data)
	lineNum := 1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return time.Time{}, nil, fmt.Errorf("failed to read line %d: %w", lineNum+1, err)
		}

		lineNum++

		if len(record) < 3 {
			return time.Time{}, nil, fmt.Errorf("line %d: expected 3 fields (family_name, kills, deaths), got %d", lineNum, len(record))
		}

		familyName := strings.TrimSpace(record[0])
		if familyName == "" {
			return time.Time{}, nil, fmt.Errorf("line %d: family_name cannot be empty", lineNum)
		}

		kills, err := strconv.Atoi(strings.TrimSpace(record[1]))
		if err != nil {
			return time.Time{}, nil, fmt.Errorf("line %d: invalid kills value '%s': %w", lineNum, record[1], err)
		}
		if kills < 0 {
			return time.Time{}, nil, fmt.Errorf("line %d: kills cannot be negative (got %d)", lineNum, kills)
		}

		deaths, err := strconv.Atoi(strings.TrimSpace(record[2]))
		if err != nil {
			return time.Time{}, nil, fmt.Errorf("line %d: invalid deaths value '%s': %w", lineNum, record[2], err)
		}
		if deaths < 0 {
			return time.Time{}, nil, fmt.Errorf("line %d: deaths cannot be negative (got %d)", lineNum, deaths)
		}

		warLines = append(warLines, db.WarLineData{
			FamilyName: familyName,
			Kills:      kills,
			Deaths:     deaths,
		})
	}

	if len(warLines) == 0 {
		return time.Time{}, nil, fmt.Errorf("no war data found in CSV")
	}

	return warDate, warLines, nil
}

// saveImage saves an uploaded image to the uploads directory
func saveImage(imageData []byte, discordUserID string, filename string) (string, error) {
	uploadsDir := "uploads"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create uploads directory: %w", err)
	}

	// Generate filename with timestamp and Discord user ID
	timestamp := time.Now().Format("20060102_150405")
	ext := filepath.Ext(filename)
	savedFilename := fmt.Sprintf("%s_%s%s", discordUserID, timestamp, ext)
	savedPath := filepath.Join(uploadsDir, savedFilename)

	// Write the file
	if err := os.WriteFile(savedPath, imageData, 0644); err != nil {
		return "", fmt.Errorf("failed to save image: %w", err)
	}

	return savedPath, nil
}

// processImageWithOpenAI sends an image to OpenAI API and extracts war data
func processImageWithOpenAI(imageData []byte, mimeType string) (warDate time.Time, warLines []db.WarLineData, err error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return time.Time{}, nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))

	// Encode image as base64
	imageBase64 := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(imageData))

	// First, check if the image passes moderation
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	
	moderationResp, err := client.Moderations.New(ctx, openai.ModerationNewParams{
		Model: openai.ModerationModelOmniModerationLatest,
		Input: openai.ModerationNewParamsInputUnion{
			OfModerationMultiModalArray: []openai.ModerationMultiModalInputUnionParam{
				openai.ModerationMultiModalInputParamOfImageURL(openai.ModerationImageURLInputImageURLParam{
					URL: imageBase64,
				}),
			},
		},
	})
	if err != nil {
		return time.Time{}, nil, fmt.Errorf("moderation API error: %w", err)
	}

	// Check if the image was flagged as unsafe
	if len(moderationResp.Results) > 0 && moderationResp.Results[0].Flagged {
		// Collect the categories that were flagged
		categories := moderationResp.Results[0].Categories
		var flaggedCategories []string
		
		if categories.Harassment {
			flaggedCategories = append(flaggedCategories, "harassment")
		}
		if categories.HarassmentThreatening {
			flaggedCategories = append(flaggedCategories, "harassment/threatening")
		}
		if categories.Hate {
			flaggedCategories = append(flaggedCategories, "hate")
		}
		if categories.HateThreatening {
			flaggedCategories = append(flaggedCategories, "hate/threatening")
		}
		if categories.Illicit {
			flaggedCategories = append(flaggedCategories, "illicit")
		}
		if categories.IllicitViolent {
			flaggedCategories = append(flaggedCategories, "illicit/violent")
		}
		if categories.SelfHarm {
			flaggedCategories = append(flaggedCategories, "self-harm")
		}
		if categories.SelfHarmInstructions {
			flaggedCategories = append(flaggedCategories, "self-harm/instructions")
		}
		if categories.SelfHarmIntent {
			flaggedCategories = append(flaggedCategories, "self-harm/intent")
		}
		if categories.Sexual {
			flaggedCategories = append(flaggedCategories, "sexual")
		}
		if categories.SexualMinors {
			flaggedCategories = append(flaggedCategories, "sexual/minors")
		}
		if categories.Violence {
			flaggedCategories = append(flaggedCategories, "violence")
		}
		if categories.ViolenceGraphic {
			flaggedCategories = append(flaggedCategories, "violence/graphic")
		}
		
		return time.Time{}, nil, fmt.Errorf("MODERATION_FAILED:%s", strings.Join(flaggedCategories, ","))
	}

	// Create the prompt for OpenAI
	prompt := "Extract the war statistics from this screenshot and return them in CSV format.\n\n" +
		"The screenshot contains war data with the following information:\n" +
		"- The date of the war is at the top of the screenshot in DD-MM-YY format (e.g., 20-03-25 for March 20, 2025)\n" +
		"- The leftmost column contains family names\n" +
		"- The last two columns (rightmost) contain kills and deaths\n" +
		"- All other columns should be ignored\n\n" +
		"IMPORTANT: The date in the screenshot is in DD-MM-YY format. You MUST return the date in the EXACT SAME DD-MM-YY format as shown in the screenshot. Do NOT convert it to any other format.\n\n" +
		"Please return the data in this exact CSV format:\n" +
		"First line: date in DD-MM-YY format (exactly as shown in the screenshot)\n" +
		"Following lines: family_name,kills,deaths\n\n" +
		"Example output:\n" +
		"20-03-25\n" +
		"FamilyName1,10,5\n" +
		"FamilyName2,15,8\n\n" +
		"CRITICAL: Return ONLY the CSV data with NO markdown formatting, NO code blocks (```), NO explanatory text, and NO additional formatting. Just the raw CSV data."

	// Now extract war stats from the image using vision API
	chatCompletion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfArrayOfContentParts: []openai.ChatCompletionContentPartUnionParam{
							openai.TextContentPart(prompt),
							openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
								URL: imageBase64,
							}),
						},
					},
				},
			},
		},
		Model:     openai.ChatModelGPT4o,
		MaxTokens: openai.Int(1000),
	})

	if err != nil {
		return time.Time{}, nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(chatCompletion.Choices) == 0 {
		return time.Time{}, nil, fmt.Errorf("no response from OpenAI API")
	}

	csvContent := chatCompletion.Choices[0].Message.Content
	
	// Validate that we got some content back
	if strings.TrimSpace(csvContent) == "" {
		return time.Time{}, nil, fmt.Errorf("OpenAI returned empty response - unable to extract war data from image")
	}
	
	// Clean the CSV content to remove markdown formatting and blank lines
	cleanedContent := cleanCSVContent(csvContent)
	
	if strings.TrimSpace(cleanedContent) == "" {
		return time.Time{}, nil, fmt.Errorf("OpenAI response contained only formatting - no actual CSV data found")
	}
	
	// Parse the cleaned CSV content returned by OpenAI
	return parseWarCSV(strings.NewReader(cleanedContent))
}

// encodeBase64 encodes data to base64 string
func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func handleAddWar(s *discordgo.Session, i *discordgo.InteractionCreate, dbx *db.DB, cfg *GuildConfig) {
	if !hasOfficerPermission(s, i, cfg) {
		discord.RespondEphemeral(s, i, "You need officer role or admin permission to use this command.")
		return
	}

	// Get the result parameter
	options := i.ApplicationCommandData().Options
	var warResult string
	for _, opt := range options {
		if opt.Name == "result" {
			warResult = opt.StringValue()
			break
		}
	}

	if warResult == "" {
		discord.RespondEphemeral(s, i, "Please select a war result (Win or Lose).")
		return
	}

	// Get the attachment
	if len(i.ApplicationCommandData().Resolved.Attachments) == 0 {
		discord.RespondText(s, i, "Please attach a CSV or image file with war data.")
		return
	}

	// Get the first attachment
	var attachment *discordgo.MessageAttachment
	for _, att := range i.ApplicationCommandData().Resolved.Attachments {
		attachment = att
		break
	}

	if attachment == nil {
		discord.RespondText(s, i, "No attachment found.")
		return
	}

	// Determine file type
	filename := strings.ToLower(attachment.Filename)
	isCSV := strings.HasSuffix(filename, ".csv")
	isImage := strings.HasSuffix(filename, ".png") || strings.HasSuffix(filename, ".jpg") || 
		strings.HasSuffix(filename, ".jpeg") || strings.HasSuffix(filename, ".webp")

	if !isCSV && !isImage {
		discord.RespondText(s, i, "File must be a CSV file (.csv) or an image file (.png, .jpg, .jpeg, .webp).")
		return
	}

	// Check file size
	maxFileSize := 10 * 1024 * 1024 // 10 MB for CSV
	if isImage {
		maxFileSize = 5 * 1024 * 1024 // 5 MB for images
	}
	if attachment.Size > maxFileSize {
		if isImage {
			discord.RespondText(s, i, "Image file size exceeds 5MB limit.")
		} else {
			discord.RespondText(s, i, "CSV file size exceeds 10MB limit.")
		}
		return
	}

	// Validate that the URL is from Discord's CDN
	if !strings.HasPrefix(attachment.URL, "https://cdn.discordapp.com/") &&
		!strings.HasPrefix(attachment.URL, "https://media.discordapp.net/") {
		log.Printf("addwar: suspicious attachment URL: %s", attachment.URL)
		discord.RespondText(s, i, "Invalid attachment source.")
		return
	}

	// Create HTTP client with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", attachment.URL, nil)
	if err != nil {
		log.Printf("addwar request creation error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to download the file. Please try again.")
		return
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Download the file
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("addwar download error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to download the file. Please try again.")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("addwar download failed with status: %d", resp.StatusCode)
		discord.RespondEphemeral(s, i, "Failed to download the file. Please try again.")
		return
	}

	// Limit the response body size as an additional safety measure
	limitedReader := io.LimitReader(resp.Body, int64(maxFileSize))
	
	// Read the file content
	fileContent, err := io.ReadAll(limitedReader)
	if err != nil {
		log.Printf("addwar read error: %v", err)
		discord.RespondEphemeral(s, i, "Failed to read the file. Please try again.")
		return
	}

	// For image processing, defer the response since it can take a while
	if isImage {
		err := discord.DeferResponse(s, i)
		if err != nil {
			log.Printf("addwar defer error: %v", err)
			// If defer fails, we can't continue as we won't be able to respond
			return
		}
	}

	var warDate time.Time
	var warLines []db.WarLineData

	if isImage {
		// Save the image locally
		savedPath, err := saveImage(fileContent, i.Member.User.ID, attachment.Filename)
		if err != nil {
			log.Printf("addwar save image error: %v", err)
			discord.FollowUpEphemeral(s, i, "Failed to save the image. Please try again.")
			return
		}
		log.Printf("Image saved to: %s", savedPath)

		// Process image with OpenAI
		warDate, warLines, err = processImageWithOpenAI(fileContent, attachment.ContentType)
		if err != nil {
			log.Printf("addwar image processing error: %v", err)
			
			// Check if this is a moderation failure
			errMsg := err.Error()
			if strings.HasPrefix(errMsg, "MODERATION_FAILED:") {
				// Extract the flagged categories
				categoriesStr := strings.TrimPrefix(errMsg, "MODERATION_FAILED:")
				categories := strings.Split(categoriesStr, ",")
				
				// Build a user-friendly message
				categoryList := strings.Join(categories, ", ")
				msg := fmt.Sprintf("⚠️ **Image Moderation Failed**\n\n"+
					"The uploaded image was flagged for potentially unsafe content.\n\n"+
					"**Flagged categories:** %s\n\n"+
					"Please upload a different image that complies with content policies.", categoryList)
				discord.FollowUpText(s, i, msg)
				return
			}
			
			// For other errors, respond ephemerally
			if len(errMsg) > 200 {
				errMsg = errMsg[:200] + "..."
			}
			discord.FollowUpEphemeral(s, i, fmt.Sprintf("Failed to process image: %s", errMsg))
			return
		}
	} else {
		// Parse CSV
		warDate, warLines, err = parseWarCSV(bytes.NewReader(fileContent))
		if err != nil {
			log.Printf("addwar parse error: %v", err)
			// Truncate error message to avoid exposing too much detail
			errMsg := err.Error()
			if len(errMsg) > 200 {
				errMsg = errMsg[:200] + "..."
			}
			discord.RespondEphemeral(s, i, fmt.Sprintf("Failed to parse CSV file: %s", errMsg))
			return
		}
	}

	// Create the war entry
	err = db.CreateWarFromCSV(dbx, i.GuildID, i.ChannelID, i.ID, i.Member.User.ID, warDate, warResult, warLines)
	if err != nil {
		log.Printf("addwar create error: %v", err)
		if isImage {
			discord.FollowUpEphemeral(s, i, "Failed to create war entry. Please try again.")
		} else {
			discord.RespondEphemeral(s, i, "Failed to create war entry. Please try again.")
		}
		return
	}

	successMsg := fmt.Sprintf("War data imported successfully!\nDate: %s\nResult: %s\nEntries: %d", warDate.Format("02-01-06"), strings.Title(warResult), len(warLines))
	if isImage {
		discord.FollowUpText(s, i, successMsg)
	} else {
		discord.RespondText(s, i, successMsg)
	}
}
