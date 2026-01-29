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

// parseWarCSV parses a CSV file with war data
// First line: date in YYYY-mm-dd format
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
	warDate, err = time.Parse("2006-01-02", strings.TrimSpace(dateRecord[0]))
	if err != nil {
		return time.Time{}, nil, fmt.Errorf("invalid date format (expected YYYY-mm-dd): %w", err)
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
	prompt := `Extract the war statistics from this screenshot and return them in CSV format.

The screenshot contains war data with the following information:
- The date of the war is at the top of the screenshot
- The leftmost column contains family names
- The last two columns (rightmost) contain kills and deaths
- All other columns should be ignored

IMPORTANT: The date MUST be returned in YYYY-MM-DD format (e.g., 2025-03-20), regardless of how it appears in the screenshot. If the date is in a different format (e.g., MM/DD/YYYY, DD-MM-YYYY), convert it to YYYY-MM-DD format.

Please return the data in this exact CSV format:
First line: date in YYYY-MM-DD format
Following lines: family_name,kills,deaths

Example output:
2025-03-20
FamilyName1,10,5
FamilyName2,15,8

Return ONLY the CSV data, no explanation or additional text.`

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
	
	// Parse the CSV content returned by OpenAI
	return parseWarCSV(strings.NewReader(csvContent))
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
		discord.RespondEphemeral(s, i, "Please attach a CSV or image file with war data.")
		return
	}

	// Get the first attachment
	var attachment *discordgo.MessageAttachment
	for _, att := range i.ApplicationCommandData().Resolved.Attachments {
		attachment = att
		break
	}

	if attachment == nil {
		discord.RespondEphemeral(s, i, "No attachment found.")
		return
	}

	// Determine file type
	filename := strings.ToLower(attachment.Filename)
	isCSV := strings.HasSuffix(filename, ".csv")
	isImage := strings.HasSuffix(filename, ".png") || strings.HasSuffix(filename, ".jpg") || 
		strings.HasSuffix(filename, ".jpeg") || strings.HasSuffix(filename, ".webp")

	if !isCSV && !isImage {
		discord.RespondEphemeral(s, i, "File must be a CSV file (.csv) or an image file (.png, .jpg, .jpeg, .webp).")
		return
	}

	// Check file size
	maxFileSize := 10 * 1024 * 1024 // 10 MB for CSV
	if isImage {
		maxFileSize = 5 * 1024 * 1024 // 5 MB for images
	}
	if attachment.Size > maxFileSize {
		if isImage {
			discord.RespondEphemeral(s, i, "Image file size exceeds 5MB limit.")
		} else {
			discord.RespondEphemeral(s, i, "CSV file size exceeds 10MB limit.")
		}
		return
	}

	// Validate that the URL is from Discord's CDN
	if !strings.HasPrefix(attachment.URL, "https://cdn.discordapp.com/") &&
		!strings.HasPrefix(attachment.URL, "https://media.discordapp.net/") {
		log.Printf("addwar: suspicious attachment URL: %s", attachment.URL)
		discord.RespondEphemeral(s, i, "Invalid attachment source.")
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

	var warDate time.Time
	var warLines []db.WarLineData

	if isImage {
		// Save the image locally
		savedPath, err := saveImage(fileContent, i.Member.User.ID, attachment.Filename)
		if err != nil {
			log.Printf("addwar save image error: %v", err)
			discord.RespondEphemeral(s, i, "Failed to save the image. Please try again.")
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
				discord.RespondText(s, i, msg)
				return
			}
			
			// For other errors, respond ephemerally
			if len(errMsg) > 200 {
				errMsg = errMsg[:200] + "..."
			}
			discord.RespondEphemeral(s, i, fmt.Sprintf("Failed to process image: %s", errMsg))
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
		discord.RespondEphemeral(s, i, "Failed to create war entry. Please try again.")
		return
	}

	discord.RespondText(s, i, fmt.Sprintf("War data imported successfully!\nDate: %s\nResult: %s\nEntries: %d", warDate.Format("2006-01-02"), strings.Title(warResult), len(warLines)))
}
