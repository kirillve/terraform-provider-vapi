package vapi

import "time"

// CreateAssistantRequest represents the request body for creating an assistant.
type CreateAssistantRequest struct {
	Name                  string            `json:"name"`
	FirstMessageMode      string            `json:"firstMessageMode,omitempty"`
	HipaaEnabled          bool              `json:"hipaaEnabled,omitempty"`
	ClientMessages        []string          `json:"clientMessages,omitempty"`
	ServerMessages        []string          `json:"serverMessages,omitempty"`
	SilenceTimeoutSeconds int               `json:"silenceTimeoutSeconds,omitempty"`
	MaxDurationSeconds    int               `json:"maxDurationSeconds,omitempty"`
	BackgroundSound       string            `json:"backgroundSound,omitempty"`
	BackgroundDenoising   bool              `json:"backgroundDenoisingEnabled,omitempty"`
	ModelOutputEnabled    bool              `json:"modelOutputInMessagesEnabled,omitempty"`
	FirstMessage          string            `json:"firstMessage,omitempty"`
	VoicemailMessage      string            `json:"voicemailMessage,omitempty"`
	EndCallMessage        string            `json:"endCallMessage,omitempty"`
	EndCallPhrases        []string          `json:"endCallPhrases,omitempty"`
	CredentialIds         []string          `json:"credentialIds,omitempty"`
	Transcriber           Transcriber       `json:"transcriber,omitempty"`
	Model                 Model             `json:"model,omitempty"`
	Voice                 Voice             `json:"voice,omitempty"`
	StartSpeakingPlan     StartSpeakingPlan `json:"startSpeakingPlan,omitempty"`
	StopSpeakingPlan      StopSpeakingPlan  `json:"stopSpeakingPlan,omitempty"`
	ServerURL             string            `json:"serverUrl,omitempty"`
	ServerURLSecret       string            `json:"serverUrlSecret,omitempty"`
	AnalysisPlan          *AnalysisPlan     `json:"analysisPlan,omitempty"`
}

// Assistant represents the response structure for an assistant.
type Assistant struct {
	ID                    string             `json:"id"`
	Name                  string             `json:"name"`
	CreatedAt             *time.Time         `json:"createdAt,omitempty"`
	UpdatedAt             *time.Time         `json:"updatedAt,omitempty"`
	FirstMessageMode      string             `json:"firstMessageMode,omitempty"`
	HipaaEnabled          bool               `json:"hipaaEnabled,omitempty"`
	ClientMessages        []string           `json:"clientMessages,omitempty"`
	ServerMessages        []string           `json:"serverMessages,omitempty"`
	SilenceTimeoutSeconds int                `json:"silenceTimeoutSeconds,omitempty"`
	MaxDurationSeconds    int                `json:"maxDurationSeconds,omitempty"`
	BackgroundSound       string             `json:"backgroundSound,omitempty"`
	BackgroundDenoising   bool               `json:"backgroundDenoisingEnabled,omitempty"`
	ModelOutputEnabled    bool               `json:"modelOutputInMessagesEnabled,omitempty"`
	FirstMessage          string             `json:"firstMessage,omitempty"`
	VoicemailMessage      string             `json:"voicemailMessage,omitempty"`
	EndCallMessage        string             `json:"endCallMessage,omitempty"`
	EndCallPhrases        []string           `json:"endCallPhrases,omitempty"`
	CredentialIds         []string           `json:"credentialIds,omitempty"`
	Transcriber           *Transcriber       `json:"transcriber,omitempty"`
	Model                 *Model             `json:"model,omitempty"`
	Voice                 *Voice             `json:"voice,omitempty"`
	StartSpeakingPlan     *StartSpeakingPlan `json:"startSpeakingPlan,omitempty"`
	StopSpeakingPlan      *StopSpeakingPlan  `json:"stopSpeakingPlan,omitempty"`
	ServerURL             string             `json:"serverUrl,omitempty"`
	ServerURLSecret       string             `json:"serverUrlSecret,omitempty"`
	AnalysisPlan          *AnalysisPlan      `json:"analysisPlan,omitempty"`
}

// Transcriber defines the transcriber settings.
type Transcriber struct {
	Provider string   `json:"provider,omitempty"`
	Model    string   `json:"model,omitempty"`
	Keywords []string `json:"keywords,omitempty"`
}

// Model defines the assistant model settings.
type Model struct {
	Model        string   `json:"model,omitempty"`
	SystemPrompt string   `json:"systemPrompt,omitempty"`
	Temperature  float64  `json:"temperature,omitempty"`
	ToolIDs      []string `json:"toolIds,omitempty"`
	Provider     string   `json:"provider,omitempty"`
	URL          string   `json:"url,omitempty"`
}

// Voice defines the voice settings for the assistant.
type Voice struct {
	Provider        string  `json:"provider,omitempty"`
	VoiceID         string  `json:"voiceId,omitempty"`
	Stability       float64 `json:"stability,omitempty"`
	SimilarityBoost float64 `json:"similarityBoost,omitempty"`
	Model           string  `json:"model,omitempty"`
}

// StartSpeakingPlan defines the configuration for starting speaking.
type StartSpeakingPlan struct {
	WaitSeconds                  float64                      `json:"waitSeconds,omitempty"`
	SmartEndpointingEnabled      bool                         `json:"smartEndpointingEnabled,omitempty"`
	TranscriptionEndpointingPlan TranscriptionEndpointingPlan `json:"transcriptionEndpointingPlan,omitempty"`
}

// TranscriptionEndpointingPlan defines the endpointing plan for transcription.
type TranscriptionEndpointingPlan struct {
	OnPunctuationSeconds   float64 `json:"onPunctuationSeconds,omitempty"`
	OnNoPunctuationSeconds float64 `json:"onNoPunctuationSeconds,omitempty"`
	OnNumberSeconds        float64 `json:"onNumberSeconds,omitempty"`
}

// StopSpeakingPlan defines the configuration for stopping speaking.
type StopSpeakingPlan struct {
	NumWords       float64  `json:"numWords,omitempty"`
	VoiceSeconds   float64  `json:"voiceSeconds,omitempty"`
	BackoffSeconds *float64 `json:"backoffSeconds,omitempty"`
}

// AnalysisPlan struct.
type AnalysisPlan struct {
	SummaryPrompt           string                `json:"summaryPrompt"`
	StructuredDataPrompt    string                `json:"structuredDataPrompt"`
	StructuredDataSchema    *StructuredDataSchema `json:"structuredDataSchema"`
	SuccessEvaluationPrompt string                `json:"successEvaluationPrompt"`
	SuccessEvaluationRubric string                `json:"successEvaluationRubric"`
}

// StructuredDataSchema struct.
type StructuredDataSchema struct {
	Type       string               `json:"type"`
	Properties map[string]*Property `json:"properties"`
}
