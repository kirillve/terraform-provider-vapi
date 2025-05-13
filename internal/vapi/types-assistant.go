package vapi

// CreateAssistantRequest represents the request body for creating an assistant.
type CreateAssistantRequest struct {
	Name                         string             `json:"name"`
	FirstMessageMode             string             `json:"firstMessageMode,omitempty"`
	HipaaEnabled                 bool               `json:"hipaaEnabled,omitempty"`
	ClientMessages               []string           `json:"clientMessages,omitempty"`
	ServerMessages               []string           `json:"serverMessages,omitempty"`
	BackgroundSound              string             `json:"backgroundSound,omitempty"`
	BackgroundDenoising          bool               `json:"backgroundDenoisingEnabled,omitempty"`
	ModelOutputEnabled           bool               `json:"modelOutputInMessagesEnabled,omitempty"`
	Language                     string             `json:"language,omitempty"`
	ForwardingPhoneNumber        string             `json:"forwardingPhoneNumber,omitempty"`
	InterruptionsEnabled         bool               `json:"interruptionsEnabled,omitempty"`
	EndCallFunctionEnabled       bool               `json:"endCallFunctionEnabled,omitempty"`
	DialKeypadFunctionEnabled    bool               `json:"dialKeypadFunctionEnabled,omitempty"`
	FillersEnabled               bool               `json:"fillersEnabled,omitempty"`
	SilenceTimeoutSeconds        *float64           `json:"silenceTimeoutSeconds,omitempty"`
	ResponseDelaySeconds         *float64           `json:"responseDelaySeconds,omitempty"`
	NumWordsToInterruptAssistant *int64             `json:"numWordsToInterruptAssistant,omitempty"`
	LiveTranscriptsEnabled       *bool              `json:"liveTranscriptsEnabled,omitempty"`
	Keywords                     []string           `json:"keywords,omitempty"`
	Voice                        *Voice             `json:"voice,omitempty"`
	Model                        *Model             `json:"model,omitempty"`
	RecordingEnabled             bool               `json:"recordingEnabled,omitempty"`
	FirstMessage                 string             `json:"firstMessage,omitempty"`
	VoicemailMessage             string             `json:"voicemailMessage,omitempty"`
	EndCallMessage               string             `json:"endCallMessage,omitempty"`
	Transcriber                  *Transcriber       `json:"transcriber,omitempty"`
	EndCallPhrases               []string           `json:"endCallPhrases,omitempty"`
	MaxDurationSeconds           int64              `json:"maxDurationSeconds,omitempty"`
	AnalysisPlan                 *AnalysisPlan      `json:"analysisPlan,omitempty"`
	MessagePlan                  *MessagePlan       `json:"messagePlan,omitempty"`
	StartSpeakingPlan            *StartSpeakingPlan `json:"startSpeakingPlan,omitempty"`
	StopSpeakingPlan             *StopSpeakingPlan  `json:"stopSpeakingPlan,omitempty"`
	Server                       *Server            `json:"server,omitempty"`
	ArtifactPlan                 *ArtifactPlan      `json:"artifactPlan,omitempty"`
}

// Assistant struct.
type Assistant struct {
	ID                           string             `json:"id,omitempty"`
	OrgID                        string             `json:"orgId,omitempty"`
	Name                         string             `json:"name,omitempty"`
	FirstMessageMode             string             `json:"firstMessageMode,omitempty"`
	HipaaEnabled                 bool               `json:"hipaaEnabled,omitempty"`
	BackgroundSound              string             `json:"backgroundSound,omitempty"`
	BackgroundDenoising          bool               `json:"backgroundDenoisingEnabled,omitempty"`
	ModelOutputEnabled           bool               `json:"modelOutputInMessagesEnabled,omitempty"`
	Language                     string             `json:"language,omitempty"`
	ForwardingPhoneNumber        string             `json:"forwardingPhoneNumber,omitempty"`
	InterruptionsEnabled         bool               `json:"interruptionsEnabled,omitempty"`
	EndCallFunctionEnabled       bool               `json:"endCallFunctionEnabled,omitempty"`
	DialKeypadFunctionEnabled    bool               `json:"dialKeypadFunctionEnabled,omitempty"`
	FillersEnabled               bool               `json:"fillersEnabled,omitempty"`
	SilenceTimeoutSeconds        *float64           `json:"silenceTimeoutSeconds,omitempty"`
	ResponseDelaySeconds         *float64           `json:"responseDelaySeconds,omitempty"`
	NumWordsToInterruptAssistant *int64             `json:"numWordsToInterruptAssistant,omitempty"`
	LiveTranscriptsEnabled       *bool              `json:"liveTranscriptsEnabled,omitempty"`
	Keywords                     []string           `json:"keywords,omitempty"`
	ParentID                     *string            `json:"parentId,omitempty"`
	Voice                        *Voice             `json:"voice,omitempty"`
	Model                        *Model             `json:"model,omitempty"`
	RecordingEnabled             bool               `json:"recordingEnabled,omitempty"`
	FirstMessage                 string             `json:"firstMessage,omitempty"`
	VoicemailMessage             string             `json:"voicemailMessage,omitempty"`
	EndCallMessage               string             `json:"endCallMessage,omitempty"`
	Transcriber                  *Transcriber       `json:"transcriber,omitempty"`
	ClientMessages               []string           `json:"clientMessages,omitempty"`
	ServerMessages               []string           `json:"serverMessages,omitempty"`
	EndCallPhrases               []string           `json:"endCallPhrases,omitempty"`
	MaxDurationSeconds           int64              `json:"maxDurationSeconds,omitempty"`
	AnalysisPlan                 *AnalysisPlan      `json:"analysisPlan,omitempty"`
	MessagePlan                  *MessagePlan       `json:"messagePlan,omitempty"`
	StartSpeakingPlan            *StartSpeakingPlan `json:"startSpeakingPlan,omitempty"`
	StopSpeakingPlan             *StopSpeakingPlan  `json:"StopSpeakingPlan,omitempty"`
	Server                       *Server            `json:"server,omitempty"`
	ArtifactPlan                 *ArtifactPlan      `json:"artifactPlan,omitempty"`
}

type ArtifactPlan struct {
	RecordingFormat string `json:"recordingFormat,omitempty"`
}

type Server struct {
	URL            string `json:"url,omitempty"`
	Secret         string `json:"secret,omitempty"`
	TimeoutSeconds int64  `json:"timeoutSeconds,omitempty"`
}

// StopSpeakingPlan struct.
type StopSpeakingPlan struct {
	NumWords       float64 `json:"numWords,omitempty"`
	VoiceSeconds   float64 `json:"voiceSeconds,omitempty"`
	BackoffSeconds float64 `json:"backoffSeconds,omitempty"`
}

// StartSpeakingPlan struct.
type StartSpeakingPlan struct {
	WaitSeconds                  float64                       `json:"waitSeconds,omitempty"`
	SmartEndpointingEnabled      bool                          `json:"smartEndpointingEnabled,omitempty"`
	TranscriptionEndpointingPlan *TranscriptionEndpointingPlan `json:"transcriptionEndpointingPlan,omitempty"`
}

// TranscriptionEndpointingPlan struct.
type TranscriptionEndpointingPlan struct {
	OnPunctuationSeconds   float64 `json:"onPunctuationSeconds,omitempty"`
	OnNoPunctuationSeconds float64 `json:"onNoPunctuationSeconds,omitempty"`
	OnNumberSeconds        float64 `json:"onNumberSeconds,omitempty"`
}

// Voice struct.
type Voice struct {
	Model           string  `json:"model,omitempty"`
	VoiceID         string  `json:"voiceId,omitempty"`
	Provider        string  `json:"provider,omitempty"`
	Stability       float64 `json:"stability,omitempty"`
	SimilarityBoost float64 `json:"similarityBoost,omitempty"`
}

// Model struct.
type Model struct {
	Model         string         `json:"model"`
	SystemPrompt  string         `json:"systemPrompt"`
	Provider      string         `json:"provider"`
	MaxTokens     int64          `json:"maxTokens,omitempty"`
	Temperature   float64        `json:"temperature,omitempty"`
	ToolIDs       []string       `json:"toolIds,omitempty"`
	Messages      []Message      `json:"messages,omitempty"`
	KnowledgeBase *KnowledgeBase `json:"knowledgeBase,omitempty"`
}

// Message struct.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// KnowledgeBase struct.
type KnowledgeBase struct {
	TopK     int64    `json:"topK"`
	FileIDs  []string `json:"fileIds"`
	Provider string   `json:"provider"`
}

// Transcriber struct.
type Transcriber struct {
	Model    string `json:"model"`
	Language string `json:"language"`
	Provider string `json:"provider"`
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

// MessagePlan struct.
type MessagePlan struct {
	IdleMessages []string `json:"idleMessages"`
}

// Customer struct.
type Customer struct {
	Number string `json:"number"`
}
